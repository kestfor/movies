package tmdb

import (
	"log/slog"
	"net/http"
	"strings"
	"sync"
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

	mu      sync.Mutex
	retryAt time.Time
	probing bool
}

const vlessRetryDelay = 5 * time.Second

func (t *failoverTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	t.mu.Lock()
	if t.probing || time.Now().Before(t.retryAt) {
		t.mu.Unlock()
		return t.direct.RoundTrip(request)
	}
	probe := !t.retryAt.IsZero()
	if probe {
		t.probing = true
	}
	t.mu.Unlock()

	response, err := t.proxy.RoundTrip(request)
	if err == nil {
		if probe {
			t.mu.Lock()
			t.probing = false
			t.retryAt = time.Time{}
			t.mu.Unlock()
		}
		return response, err
	}
	if request.Context().Err() != nil {
		if probe {
			t.mu.Lock()
			t.probing = false
			t.mu.Unlock()
		}
		return response, err
	}

	t.mu.Lock()
	if probe || t.retryAt.IsZero() {
		t.retryAt = time.Now().Add(vlessRetryDelay)
		t.logger.Warn("VLESS VPN connection failed; falling back to direct TMDB connection", "retry_after", vlessRetryDelay)
	}
	if probe {
		t.probing = false
	}
	t.mu.Unlock()
	return t.direct.RoundTrip(request)
}
