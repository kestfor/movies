package tmdb

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
)

const testVLESSURL = "vless://11111111-1111-4111-8111-111111111111@203.0.113.10:443?mode=auto&path=%2Ftelegram&security=reality&encryption=none&pbk=AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA&host=example-host&fp=chrome&type=xhttp&sni=example.com&sid=0011223344556677"

func TestNewConnectionUsesVLESSWhenConfigured(t *testing.T) {
	connection := NewConnection(testVLESSURL, slog.New(slog.NewTextHandler(io.Discard, nil)))
	t.Cleanup(func() {
		if connection.instance != nil && connection.instance.IsRunning() {
			_ = connection.Close()
		}
	})

	if connection.instance == nil || !connection.instance.IsRunning() {
		t.Fatal("VLESS instance did not start")
	}
	transport, ok := connection.Client.Transport.(*failoverTransport)
	if !ok || transport.proxy.(*http.Transport).DialContext == nil {
		t.Fatal("TMDB client does not use the VLESS dialer")
	}
	if err := connection.Close(); err != nil {
		t.Fatalf("close VLESS instance: %v", err)
	}
	if connection.instance.IsRunning() {
		t.Fatal("VLESS instance is still running after close")
	}
}

func TestNewConnectionUsesDirectTransportWithoutValidVLESSURL(t *testing.T) {
	for _, rawURL := range []string{"", "invalid"} {
		t.Run(rawURL, func(t *testing.T) {
			connection := NewConnection(rawURL, slog.New(slog.NewTextHandler(io.Discard, nil)))
			defer connection.Close()
			if connection.instance != nil {
				t.Fatal("unexpected VLESS instance")
			}
			if _, ok := connection.Client.Transport.(*http.Transport); !ok {
				t.Fatal("TMDB client does not use a direct transport")
			}
		})
	}
}

func TestFailoverTransportRetriesConnectionErrorAndStaysDirect(t *testing.T) {
	var proxyCalls, directCalls int
	transport := &failoverTransport{
		proxy: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			proxyCalls++
			return nil, errors.New("proxy connection failed")
		}),
		direct: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			directCalls++
			return jsonResponse(map[string]any{"ok": true}), nil
		}),
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	client := &http.Client{Transport: transport}
	for range 2 {
		response, err := client.Get("https://api.themoviedb.org/3/search/multi")
		if err != nil {
			t.Fatalf("TMDB request failed: %v", err)
		}
		response.Body.Close()
	}
	if proxyCalls != 1 || directCalls != 2 {
		t.Fatalf("proxy calls = %d, direct calls = %d; want 1 and 2", proxyCalls, directCalls)
	}
}

func TestFailoverTransportKeepsHTTPResponse(t *testing.T) {
	var directCalls int
	transport := &failoverTransport{
		proxy: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusBadGateway, Body: io.NopCloser(strings.NewReader("upstream error")), Header: make(http.Header)}, nil
		}),
		direct: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			directCalls++
			return jsonResponse(nil), nil
		}),
		logger: slog.Default(),
	}
	response, err := (&http.Client{Transport: transport}).Get("https://api.themoviedb.org/3/search/multi")
	if err != nil {
		t.Fatalf("TMDB request failed: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusBadGateway || directCalls != 0 {
		t.Fatalf("status = %d, direct calls = %d; want 502 and 0", response.StatusCode, directCalls)
	}
}

func TestFailoverTransportDoesNotRetryCanceledRequest(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var directCalls int
	transport := &failoverTransport{
		proxy: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			cancel()
			return nil, context.Canceled
		}),
		direct: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			directCalls++
			return jsonResponse(nil), nil
		}),
		logger: slog.Default(),
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.themoviedb.org/3/search/multi", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = (&http.Client{Transport: transport}).Do(request)
	if !errors.Is(err, context.Canceled) || directCalls != 0 || transport.failed.Load() {
		t.Fatalf("error = %v, direct calls = %d, failed = %t", err, directCalls, transport.failed.Load())
	}
}
