package backup

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/url"
	"strings"
	"testing"

	"github.com/mymmrac/telego/telegoapi"
)

const testVLESSURL = "vless://11111111-1111-4111-8111-111111111111@203.0.113.10:443?mode=auto&path=%2Ftelegram&security=reality&encryption=none&extra=%7BscMaxEachPostBytes%3D1000000%2C%20xPaddingBytes%3D100-1000%7D&pbk=AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA&host=example-host&fp=chrome&spx=%2Fprobe&type=xhttp&sni=example.com&sid=0011223344556677#test"

func TestFailoverCallerRetriesStreamDirectly(t *testing.T) {
	t.Parallel()

	var logOutput bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logOutput, nil))
	var proxyCalls int
	var directCalls int
	var proxyBody string
	var directBody string

	caller := &failoverCaller{
		proxy: callerFunc(func(_ context.Context, _ string, data *telegoapi.RequestData) (*telegoapi.Response, error) {
			proxyCalls++
			proxyBody = readRequestBody(t, data)
			return nil, &url.Error{Op: "Post", URL: "https://api.telegram.org", Err: io.ErrUnexpectedEOF}
		}),
		direct: callerFunc(func(_ context.Context, _ string, data *telegoapi.RequestData) (*telegoapi.Response, error) {
			directCalls++
			directBody = readRequestBody(t, data)
			return &telegoapi.Response{Ok: true}, nil
		}),
		logger: logger,
	}

	response, err := caller.Call(context.Background(), "https://api.telegram.org", &telegoapi.RequestData{
		ContentType: "multipart/form-data",
		BodyStream:  strings.NewReader("backup-body"),
	})
	if err != nil {
		t.Fatalf("Call returned error: %v", err)
	}
	if response == nil || !response.Ok {
		t.Fatalf("unexpected response: %+v", response)
	}
	if proxyCalls != 1 || directCalls != 1 {
		t.Fatalf("unexpected call counts: proxy=%d direct=%d", proxyCalls, directCalls)
	}
	if proxyBody != "backup-body" || directBody != proxyBody {
		t.Fatalf("request was not replayed: proxy=%q direct=%q", proxyBody, directBody)
	}
	if !strings.Contains(logOutput.String(), "falling back to direct Telegram connection") {
		t.Fatalf("fallback warning was not logged: %s", logOutput.String())
	}

	_, err = caller.Call(context.Background(), "https://api.telegram.org", &telegoapi.RequestData{BodyRaw: []byte("next")})
	if err != nil {
		t.Fatalf("second Call returned error: %v", err)
	}
	if proxyCalls != 1 || directCalls != 2 {
		t.Fatalf("caller did not stay on direct connection: proxy=%d direct=%d", proxyCalls, directCalls)
	}
}

func TestFailoverCallerDoesNotRetryApplicationError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("telegram API error")
	directCalled := false
	caller := &failoverCaller{
		proxy: callerFunc(func(context.Context, string, *telegoapi.RequestData) (*telegoapi.Response, error) {
			return nil, wantErr
		}),
		direct: callerFunc(func(context.Context, string, *telegoapi.RequestData) (*telegoapi.Response, error) {
			directCalled = true
			return nil, nil
		}),
		logger: slog.Default(),
	}

	_, err := caller.Call(context.Background(), "https://api.telegram.org", &telegoapi.RequestData{BodyRaw: []byte("body")})
	if !errors.Is(err, wantErr) {
		t.Fatalf("unexpected error: %v", err)
	}
	if directCalled {
		t.Fatal("direct caller was used for an application error")
	}
}

type callerFunc func(context.Context, string, *telegoapi.RequestData) (*telegoapi.Response, error)

func (f callerFunc) Call(
	ctx context.Context,
	requestURL string,
	data *telegoapi.RequestData,
) (*telegoapi.Response, error) {
	return f(ctx, requestURL, data)
}

func readRequestBody(t *testing.T, data *telegoapi.RequestData) string {
	t.Helper()
	if data.BodyRaw != nil {
		return string(data.BodyRaw)
	}
	body, err := io.ReadAll(data.BodyStream)
	if err != nil {
		t.Fatalf("read request body: %v", err)
	}
	return string(body)
}
