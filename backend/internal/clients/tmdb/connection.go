package tmdb

import (
	"log/slog"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"movies/backend/internal/network/vless"

	xraycore "github.com/xtls/xray-core/core"
)

// Connection owns the HTTP transports and optional VLESS instance used by TMDB.
type Connection struct {
	Client     *http.Client
	instance   *xraycore.Instance
	transports []*http.Transport
}

func NewConnection(rawVLESSURL string, logger *slog.Logger) *Connection {
	if logger == nil {
		logger = slog.Default()
	}

	directTransport := http.DefaultTransport.(*http.Transport).Clone()
	connection := &Connection{
		Client:     &http.Client{Timeout: 10 * time.Second, Transport: directTransport},
		transports: []*http.Transport{directTransport},
	}
	if strings.TrimSpace(rawVLESSURL) == "" {
		return connection
	}

	instance, err := vless.Start(rawVLESSURL)
	if err != nil {
		logger.Warn("failed to configure BACKUP_VLESS_URL; using direct TMDB connection", "error", err)
		return connection
	}

	proxyTransport := http.DefaultTransport.(*http.Transport).Clone()
	proxyTransport.Proxy = nil
	proxyTransport.DialContext = vless.DialContext(instance)
	connection.Client.Transport = &failoverTransport{
		proxy:  proxyTransport,
		direct: directTransport,
		logger: logger,
	}
	connection.instance = instance
	connection.transports = append(connection.transports, proxyTransport)
	logger.Info("TMDB VLESS VPN configured")
	return connection
}

func (c *Connection) Close() error {
	for _, transport := range c.transports {
		transport.CloseIdleConnections()
	}
	if c.instance != nil {
		return c.instance.Close()
	}
	return nil
}

type failoverTransport struct {
	proxy  http.RoundTripper
	direct http.RoundTripper
	logger *slog.Logger
	failed atomic.Bool
}

func (t *failoverTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if t.failed.Load() {
		return t.direct.RoundTrip(request)
	}

	response, err := t.proxy.RoundTrip(request)
	if err == nil || request.Context().Err() != nil {
		return response, err
	}

	if t.failed.CompareAndSwap(false, true) {
		t.logger.Warn("VLESS VPN connection failed; falling back to direct TMDB connection")
	}
	return t.direct.RoundTrip(request)
}
