package backup

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/mymmrac/telego/telegoapi"
)

func TestTelegramClientRetriesSendsOnNewConnection(t *testing.T) {
	for _, method := range []string{"sendMessage", "sendDocument"} {
		for _, failure := range []string{"success", "transport", "api", "retry fails", "canceled"} {
			t.Run(method+"/"+failure, func(t *testing.T) {
				tmpDir := t.TempDir()
				t.Setenv("TMPDIR", tmpDir)
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				calls, created := 0, 0
				wantErr := errors.New("connection failed")
				var previous *TelegramConnection
				client := &TelegramClient{logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
				client.newConnection = func() *TelegramConnection {
					if previous != nil && previous.instance.IsRunning() {
						t.Fatal("old VLESS instance still running during reconnect")
					}
					created++
					generation := created
					connection := NewTelegramConnection(testVLESSURL, client.logger)
					if connection.instance == nil || !connection.instance.IsRunning() {
						t.Fatal("VLESS instance did not start")
					}
					t.Cleanup(func() { _ = connection.Close() })
					previous = connection
					connection.Caller = callerFunc(func(_ context.Context, _ string, data *telegoapi.RequestData) (*telegoapi.Response, error) {
						calls++
						if body := readRequestBody(t, data); body != "complete request body" {
							t.Fatalf("request body was not replayed: %q", body)
						}
						if generation == 1 {
							switch failure {
							case "success":
								return &telegoapi.Response{Ok: true}, nil
							case "api":
								return &telegoapi.Response{Error: &telegoapi.Error{ErrorCode: 502}}, nil
							case "canceled":
								cancel()
								return nil, context.Canceled
							default:
								return nil, wantErr
							}
						}
						if failure == "retry fails" {
							return nil, wantErr
						}
						return &telegoapi.Response{Ok: true}, nil
					})
					return connection
				}
				client.connection = client.newConnection()
				defer client.Close()
				data := &telegoapi.RequestData{BodyRaw: []byte("complete request body")}
				if method == "sendDocument" {
					data = &telegoapi.RequestData{BodyStream: strings.NewReader("complete request body")}
				}
				response, err := client.Call(ctx, "https://api.telegram.org/bottest/"+method, data)
				wantCalls := 2
				if failure == "success" {
					wantCalls = 1
				}
				switch failure {
				case "canceled":
					wantCalls = 1
					if !errors.Is(err, context.Canceled) {
						t.Fatalf("unexpected cancellation error: %v", err)
					}
				case "retry fails":
					if !errors.Is(err, wantErr) {
						t.Fatalf("unexpected retry error: %v", err)
					}
				default:
					if err != nil || response == nil || !response.Ok {
						t.Fatalf("send did not recover: response=%v err=%v", response, err)
					}
				}
				if calls != wantCalls || created != wantCalls {
					t.Fatalf("calls=%d connections=%d; want %d", calls, created, wantCalls)
				}
				if files, err := os.ReadDir(tmpDir); err != nil || len(files) != 0 {
					t.Fatalf("request buffer was not cleaned up: files=%v err=%v", files, err)
				}
				if err := client.Close(); err != nil {
					t.Fatal(err)
				}
				if previous.instance.IsRunning() {
					t.Fatal("last VLESS instance still running after close")
				}
				client.Reconnect(context.Background())
				if created != wantCalls {
					t.Fatal("closed client reconnected")
				}
				if _, err := client.Call(ctx, "https://api.telegram.org/bottest/getUpdates", data); !errors.Is(err, net.ErrClosed) {
					t.Fatalf("closed client returned %v", err)
				}
			})
		}
	}
}

func TestTelegramClientReconnectWaitsForActiveSend(t *testing.T) {
	started, release, sent, reconnected := make(chan struct{}), make(chan struct{}), make(chan struct{}), make(chan struct{})
	old := &TelegramConnection{Caller: callerFunc(func(context.Context, string, *telegoapi.RequestData) (*telegoapi.Response, error) {
		close(started)
		<-release
		return &telegoapi.Response{Ok: true}, nil
	})}
	client := &TelegramClient{connection: old, logger: slog.Default(), newConnection: func() *TelegramConnection {
		return &TelegramConnection{}
	}}
	defer client.Close()
	go func() {
		defer close(sent)
		if _, err := client.Call(context.Background(), "https://api.telegram.org/bottest/sendDocument", &telegoapi.RequestData{BodyRaw: []byte("body")}); err != nil {
			t.Errorf("send failed: %v", err)
		}
	}()
	<-started
	go func() {
		client.Reconnect(context.Background())
		close(reconnected)
	}()
	select {
	case <-reconnected:
		t.Fatal("reconnected while send was active")
	case <-time.After(50 * time.Millisecond):
	}
	close(release)
	<-sent
	<-reconnected
	if client.connection == old {
		t.Fatal("connection was not replaced")
	}
}
