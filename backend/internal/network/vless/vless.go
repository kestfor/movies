package vless

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	stdnet "net"
	"net/url"
	"strconv"
	"strings"

	_ "github.com/xtls/xray-core/app/dispatcher"
	_ "github.com/xtls/xray-core/app/log"
	_ "github.com/xtls/xray-core/app/proxyman/inbound"
	_ "github.com/xtls/xray-core/app/proxyman/outbound"
	xraynet "github.com/xtls/xray-core/common/net"
	xraycore "github.com/xtls/xray-core/core"
	_ "github.com/xtls/xray-core/main/json"
	_ "github.com/xtls/xray-core/proxy/vless/outbound"
	_ "github.com/xtls/xray-core/transport/internet/grpc"
	_ "github.com/xtls/xray-core/transport/internet/reality"
	_ "github.com/xtls/xray-core/transport/internet/splithttp"
)

func Start(rawURL string) (*xraycore.Instance, error) {
	config, err := parseVLESSURL(rawURL)
	if err != nil {
		return nil, fmt.Errorf("parse VLESS URL: %w", err)
	}
	rawConfig, err := config.xrayConfig()
	if err != nil {
		return nil, fmt.Errorf("prepare VLESS VPN: %w", err)
	}
	instance, err := xraycore.StartInstance("json", rawConfig)
	if err != nil {
		return nil, fmt.Errorf("start VLESS VPN: %w", err)
	}
	return instance, nil
}

type vlessConnection struct {
	Address     string
	Port        uint16
	ID          string
	Encryption  string
	Flow        string
	Transport   string
	Mode        string
	Path        string
	Host        string
	ServiceName string
	Authority   string
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
	transport := strings.ToLower(query.Get("type"))
	if transport != "xhttp" && transport != "grpc" {
		return vlessConnection{}, errors.New("only VLESS XHTTP and gRPC transports are supported")
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
	path := query.Get("path")
	serviceName := query.Get("serviceName")
	var extra map[string]any
	switch transport {
	case "xhttp":
		if mode == "" {
			mode = "auto"
		}
		switch mode {
		case "auto", "packet-up", "stream-up", "stream-one":
		default:
			return vlessConnection{}, errors.New("unsupported VLESS XHTTP mode")
		}
		if path == "" {
			path = "/"
		}
		extra, err = parseXHTTPExtra(query.Get("extra"))
		if err != nil {
			return vlessConnection{}, err
		}
	case "grpc":
		if serviceName == "" {
			return vlessConnection{}, errors.New("VLESS gRPC serviceName is required")
		}
		if mode == "" {
			mode = "gun"
		}
		if mode != "gun" && mode != "multi" {
			return vlessConnection{}, errors.New("unsupported VLESS gRPC mode")
		}
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

	return vlessConnection{
		Address:     parsed.Hostname(),
		Port:        uint16(port),
		ID:          parsed.User.Username(),
		Encryption:  encryption,
		Flow:        query.Get("flow"),
		Transport:   transport,
		Mode:        mode,
		Path:        path,
		Host:        query.Get("host"),
		ServiceName: serviceName,
		Authority:   query.Get("authority"),
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
	streamSettings := map[string]any{
		"network":  c.Transport,
		"security": "reality",
		"realitySettings": map[string]any{
			"serverName":  c.ServerName,
			"fingerprint": c.Fingerprint,
			"publicKey":   c.PublicKey,
			"shortId":     c.ShortID,
			"spiderX":     c.SpiderX,
		},
	}
	if c.Transport == "grpc" {
		streamSettings["grpcSettings"] = map[string]any{
			"serviceName": c.ServiceName,
			"authority":   c.Authority,
			"multiMode":   c.Mode == "multi",
		}
	} else {
		xhttpSettings := map[string]any{
			"host": c.Host,
			"path": c.Path,
			"mode": c.Mode,
		}
		if len(c.Extra) > 0 {
			xhttpSettings["extra"] = c.Extra
		}
		streamSettings["xhttpSettings"] = xhttpSettings
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
				"streamSettings": streamSettings,
			},
		},
	}

	return json.Marshal(config)
}

func DialContext(instance *xraycore.Instance) func(context.Context, string, string) (stdnet.Conn, error) {
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
