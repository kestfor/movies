package backup

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync/atomic"

	"movies/backend/internal/network/vless"

	"github.com/mymmrac/telego/telegoapi"
	xraycore "github.com/xtls/xray-core/core"
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

	instance, err := vless.Start(rawVLESSURL)
	if err != nil {
		logger.Warn("failed to configure BACKUP_VLESS_URL; using direct Telegram connection", "error", err)
		return connection
	}

	proxyTransport := http.DefaultTransport.(*http.Transport).Clone()
	proxyTransport.Proxy = nil
	proxyTransport.DialContext = vless.DialContext(instance)
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
