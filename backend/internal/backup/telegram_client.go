package backup

import (
	"context"
	"log/slog"
	"net"
	"strings"
	"sync"

	"github.com/mymmrac/telego/telegoapi"
)

// TelegramClient keeps the bot and backup sender on the current connection.
type TelegramClient struct {
	mu            sync.RWMutex
	connection    *TelegramConnection
	newConnection func() *TelegramConnection
	logger        *slog.Logger
}

func NewTelegramClient(vlessURL string, logger *slog.Logger) *TelegramClient {
	if logger == nil {
		logger = slog.Default()
	}
	newConnection := func() *TelegramConnection {
		return NewTelegramConnection(vlessURL, logger)
	}
	return &TelegramClient{connection: newConnection(), newConnection: newConnection, logger: logger}
}

func (c *TelegramClient) Call(ctx context.Context, requestURL string, data *telegoapi.RequestData) (*telegoapi.Response, error) {
	if !strings.HasSuffix(requestURL, "/sendMessage") && !strings.HasSuffix(requestURL, "/sendDocument") {
		return c.call(ctx, requestURL, data)
	}

	replay, err := prepareReplayableRequest(data)
	if err != nil {
		return nil, err
	}
	defer replay.Close()
	response, err := c.call(ctx, requestURL, replay.Data())
	if ctx.Err() != nil || (err == nil && response != nil && response.Ok) {
		return response, err
	}
	c.logger.Warn("telegram backup bot send failed; reconnecting")
	c.Reconnect(ctx)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return c.call(ctx, requestURL, replay.Data())
}

func (c *TelegramClient) call(ctx context.Context, requestURL string, data *telegoapi.RequestData) (*telegoapi.Response, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.connection == nil {
		return nil, net.ErrClosed
	}
	return c.connection.Caller.Call(ctx, requestURL, data)
}

func (c *TelegramClient) Reconnect(ctx context.Context) {
	// Wait for in-flight sends before closing their transport and VLESS instance.
	c.mu.Lock()
	defer c.mu.Unlock()
	if ctx.Err() != nil || c.connection == nil {
		return
	}
	if err := c.connection.Close(); err != nil {
		c.logger.Warn("close Telegram connection", "error", err)
	}
	c.connection = c.newConnection()
	c.logger.Info("telegram backup bot connection recreated")
}

func (c *TelegramClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.connection == nil {
		return nil
	}
	err := c.connection.Close()
	c.connection = nil
	return err
}
