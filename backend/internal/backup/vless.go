package backup

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	stdnet "net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/mymmrac/telego/telegoapi"
	_ "github.com/xtls/xray-core/app/dispatcher"
	_ "github.com/xtls/xray-core/app/log"
	_ "github.com/xtls/xray-core/app/proxyman/inbound"
	_ "github.com/xtls/xray-core/app/proxyman/outbound"
	xraynet "github.com/xtls/xray-core/common/net"
	xraycore "github.com/xtls/xray-core/core"
	_ "github.com/xtls/xray-core/main/json"
	_ "github.com/xtls/xray-core/proxy/vless/outbound"
	_ "github.com/xtls/xray-core/transport/internet/reality"
	_ "github.com/xtls/xray-core/transport/internet/splithttp"
)

type TelegramConnection struct {
	Caller     telegoapi.Caller
	instance   *xraycore.Instance
	transports []*http.Transport
}

func NewTelegramConnection(rawVLESSURL string, logger *slog.Logger) *TelegramConnection {
	if logger == nil {
		logger = slog.Default()
	}

	directTransport := http.DefaultTransport.(*http.Transport).Clone()
	directCaller := telegoapi.HTTPCaller{Client: &http.Client{Transport: directTransport}}
	connection := &TelegramConnection{
		Caller:     directCaller,
		transports: []*http.Transport{directTransport},
	}

	if strings.TrimSpace(rawVLESSURL) == "" {
		logger.Warn("BACKUP_VLESS_URL is empty; using direct Telegram connection")
		return connection
	}

	vless, err := parseVLESSURL(rawVLESSURL)
	if err != nil {
		logger.Warn("invalid BACKUP_VLESS_URL; using direct Telegram connection")
		return connection
	}

	config, err := vless.xrayConfig()
	if err != nil {
		logger.Warn("failed to prepare VLESS VPN; using direct Telegram connection")
		return connection
	}

	instance, err := xraycore.StartInstance("json", config)
	if err != nil {
		logger.Warn("failed to start VLESS VPN; using direct Telegram connection")
		return connection
	}

	proxyTransport := http.DefaultTransport.(*http.Transport).Clone()
	proxyTransport.DialContext = xrayDialContext(instance)
	proxyCaller := telegoapi.HTTPCaller{Client: &http.Client{Transport: proxyTransport}}

	connection.Caller = &failoverCaller{
		proxy:  proxyCaller,
		direct: directCaller,
		logger: logger,
	}
	connection.instance = instance
	connection.transports = append(connection.transports, proxyTransport)
	logger.Info("Telegram VLESS VPN configured")

	return connection
}

func (c *TelegramConnection) Close() error {
	for _, transport := range c.transports {
		transport.CloseIdleConnections()
	}
	if c.instance != nil {
		return c.instance.Close()
	}
	return nil
}

type vlessConnection struct {
	Address     string
	Port        uint16
	ID          string
	Encryption  string
	Flow        string
	Mode        string
	Path        string
	Host        string
	ServerName  string
	Fingerprint string
	PublicKey   string
	ShortID     string
	SpiderX     string
	Extra       map[string]any
}

func parseVLESSURL(raw string) (vlessConnection, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || !strings.EqualFold(parsed.Scheme, "vless") {
		return vlessConnection{}, errors.New("invalid VLESS URL")
	}
	if parsed.User == nil || parsed.User.Username() == "" {
		return vlessConnection{}, errors.New("VLESS user ID is required")
	}
	if parsed.Hostname() == "" || parsed.Port() == "" {
		return vlessConnection{}, errors.New("VLESS server and port are required")
	}

	port, err := strconv.ParseUint(parsed.Port(), 10, 16)
	if err != nil || port == 0 {
		return vlessConnection{}, errors.New("invalid VLESS port")
	}

	query := parsed.Query()
	if !strings.EqualFold(query.Get("type"), "xhttp") {
		return vlessConnection{}, errors.New("only VLESS XHTTP transport is supported")
	}
	if !strings.EqualFold(query.Get("security"), "reality") {
		return vlessConnection{}, errors.New("only VLESS REALITY security is supported")
	}

	encryption := query.Get("encryption")
	if encryption == "" {
		encryption = "none"
	}
	if encryption != "none" {
		return vlessConnection{}, errors.New("unsupported VLESS encryption")
	}

	mode := query.Get("mode")
	if mode == "" {
		mode = "auto"
	}
	switch mode {
	case "auto", "packet-up", "stream-up", "stream-one":
	default:
		return vlessConnection{}, errors.New("unsupported VLESS XHTTP mode")
	}

	path := query.Get("path")
	if path == "" {
		path = "/"
	}
	spiderX := query.Get("spx")
	if spiderX == "" {
		spiderX = "/"
	}

	publicKey := query.Get("pbk")
	decodedPublicKey, err := base64.RawURLEncoding.DecodeString(publicKey)
	if err != nil || len(decodedPublicKey) != 32 {
		return vlessConnection{}, errors.New("invalid VLESS REALITY public key")
	}
	shortID := query.Get("sid")
	if len(shortID) > 16 || len(shortID)%2 != 0 {
		return vlessConnection{}, errors.New("invalid VLESS REALITY short ID")
	}
	if _, err := hex.DecodeString(shortID); err != nil {
		return vlessConnection{}, errors.New("invalid VLESS REALITY short ID")
	}
	if query.Get("sni") == "" || query.Get("fp") == "" {
		return vlessConnection{}, errors.New("VLESS REALITY SNI and fingerprint are required")
	}

	extra, err := parseXHTTPExtra(query.Get("extra"))
	if err != nil {
		return vlessConnection{}, err
	}

	return vlessConnection{
		Address:     parsed.Hostname(),
		Port:        uint16(port),
		ID:          parsed.User.Username(),
		Encryption:  encryption,
		Flow:        query.Get("flow"),
		Mode:        mode,
		Path:        path,
		Host:        query.Get("host"),
		ServerName:  query.Get("sni"),
		Fingerprint: query.Get("fp"),
		PublicKey:   publicKey,
		ShortID:     shortID,
		SpiderX:     spiderX,
		Extra:       extra,
	}, nil
}

func parseXHTTPExtra(raw string) (map[string]any, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	var extra map[string]any
	if json.Unmarshal([]byte(raw), &extra) == nil {
		return extra, nil
	}

	raw = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(raw, "{"), "}"))
	if raw == "" {
		return nil, nil
	}

	extra = make(map[string]any)
	for _, field := range strings.Split(raw, ",") {
		key, value, ok := strings.Cut(field, "=")
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if !ok || key == "" || value == "" {
			return nil, errors.New("invalid VLESS XHTTP extra settings")
		}
		extra[key] = parseExtraValue(value)
	}
	return extra, nil
}

func parseExtraValue(value string) any {
	if parsed, err := strconv.ParseInt(value, 10, 64); err == nil {
		return parsed
	}
	if parsed, err := strconv.ParseBool(value); err == nil {
		return parsed
	}
	return strings.Trim(value, "\"'")
}

func (c vlessConnection) xrayConfig() ([]byte, error) {
	xhttpSettings := map[string]any{
		"host": c.Host,
		"path": c.Path,
		"mode": c.Mode,
	}
	if len(c.Extra) > 0 {
		xhttpSettings["extra"] = c.Extra
	}

	config := map[string]any{
		"log": map[string]any{
			"loglevel": "none",
		},
		"outbounds": []any{
			map[string]any{
				"protocol": "vless",
				"settings": map[string]any{
					"address":    c.Address,
					"port":       c.Port,
					"id":         c.ID,
					"encryption": c.Encryption,
					"flow":       c.Flow,
				},
				"streamSettings": map[string]any{
					"network":  "xhttp",
					"security": "reality",
					"realitySettings": map[string]any{
						"serverName":  c.ServerName,
						"fingerprint": c.Fingerprint,
						"publicKey":   c.PublicKey,
						"shortId":     c.ShortID,
						"spiderX":     c.SpiderX,
					},
					"xhttpSettings": xhttpSettings,
				},
			},
		},
	}

	return json.Marshal(config)
}

func xrayDialContext(instance *xraycore.Instance) func(context.Context, string, string) (stdnet.Conn, error) {
	return func(ctx context.Context, network, address string) (stdnet.Conn, error) {
		if !strings.HasPrefix(network, "tcp") {
			return nil, fmt.Errorf("unsupported network %q", network)
		}
		host, rawPort, err := stdnet.SplitHostPort(address)
		if err != nil {
			return nil, fmt.Errorf("split destination: %w", err)
		}
		port, err := strconv.ParseUint(rawPort, 10, 16)
		if err != nil || port == 0 {
			return nil, errors.New("invalid destination port")
		}
		destination := xraynet.TCPDestination(xraynet.ParseAddress(host), xraynet.Port(port))
		return xraycore.Dial(ctx, instance, destination)
	}
}

type failoverCaller struct {
	proxy  telegoapi.Caller
	direct telegoapi.Caller
	logger *slog.Logger
	failed atomic.Bool
}

func (c *failoverCaller) Call(
	ctx context.Context,
	requestURL string,
	data *telegoapi.RequestData,
) (*telegoapi.Response, error) {
	if c.failed.Load() {
		return c.direct.Call(ctx, requestURL, data)
	}

	replay, err := prepareReplayableRequest(data)
	if err != nil {
		return nil, err
	}
	defer replay.Close()

	response, err := c.proxy.Call(ctx, requestURL, replay.Data())
	if err == nil || !isConnectionError(err) || ctx.Err() != nil {
		return response, err
	}

	if c.failed.CompareAndSwap(false, true) {
		c.logger.Warn("VLESS VPN connection failed; falling back to direct Telegram connection")
	}
	return c.direct.Call(ctx, requestURL, replay.Data())
}

func isConnectionError(err error) bool {
	var urlError *url.Error
	return errors.As(err, &urlError)
}

type replayableRequest struct {
	contentType string
	raw         []byte
	file        *os.File
	size        int64
}

func prepareReplayableRequest(data *telegoapi.RequestData) (*replayableRequest, error) {
	if data == nil {
		return nil, errors.New("Telegram request data is nil")
	}
	if data.BodyRaw != nil {
		return &replayableRequest{contentType: data.ContentType, raw: data.BodyRaw}, nil
	}
	if data.BodyStream == nil {
		return nil, errors.New("Telegram request body is missing")
	}

	file, err := os.CreateTemp("", "movies-telegram-request-*")
	if err != nil {
		return nil, fmt.Errorf("create Telegram request buffer: %w", err)
	}
	size, err := io.Copy(file, data.BodyStream)
	if err != nil {
		_ = file.Close()
		_ = os.Remove(file.Name())
		return nil, fmt.Errorf("buffer Telegram request: %w", err)
	}

	return &replayableRequest{contentType: data.ContentType, file: file, size: size}, nil
}

func (r *replayableRequest) Data() *telegoapi.RequestData {
	data := &telegoapi.RequestData{ContentType: r.contentType}
	if r.raw != nil {
		data.BodyRaw = r.raw
	} else {
		data.BodyStream = io.NewSectionReader(r.file, 0, r.size)
	}
	return data
}

func (r *replayableRequest) Close() {
	if r.file == nil {
		return
	}
	name := r.file.Name()
	_ = r.file.Close()
	_ = os.Remove(name)
}
