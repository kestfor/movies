package tmdb

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
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

func TestFailoverTransportRetriesVLESSAfterCooldown(t *testing.T) {
	var proxyCalls, directCalls int
	transport := &failoverTransport{
		proxy: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			proxyCalls++
			if proxyCalls <= 2 {
				return nil, errors.New("proxy connection failed")
			}
			return jsonResponse(map[string]any{"ok": true}), nil
		}),
		direct: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			directCalls++
			return jsonResponse(map[string]any{"ok": true}), nil
		}),
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	client := &http.Client{Transport: transport}
	request := func() {
		t.Helper()
		response, err := client.Get("https://api.themoviedb.org/3/search/multi")
		if err != nil {
			t.Fatalf("TMDB request failed: %v", err)
		}
		response.Body.Close()
	}
	beforeFailure := time.Now()
	request() // VLESS fails, then direct handles the same request.
	if transport.retryAt.Before(beforeFailure.Add(vlessRetryDelay)) || transport.retryAt.After(time.Now().Add(vlessRetryDelay)) {
		t.Fatalf("retry at = %v; want five seconds after VLESS failure", transport.retryAt)
	}
	request() // Direct is used during cooldown.
	if proxyCalls != 1 || directCalls != 2 {
		t.Fatalf("during cooldown: proxy calls = %d, direct calls = %d; want 1 and 2", proxyCalls, directCalls)
	}
	transport.retryAt = time.Now().Add(-time.Second)
	request() // The first probe fails and starts another cooldown.
	request()
	if proxyCalls != 2 || directCalls != 4 {
		t.Fatalf("after failed probe: proxy calls = %d, direct calls = %d; want 2 and 4", proxyCalls, directCalls)
	}
	transport.retryAt = time.Now().Add(-time.Second)
	request() // A successful probe restores VLESS.
	request()
	if proxyCalls != 4 || directCalls != 4 || !transport.retryAt.IsZero() {
		t.Fatalf("after successful probe: proxy calls = %d, direct calls = %d, retry at = %v; want 4, 4, zero", proxyCalls, directCalls, transport.retryAt)
	}
}

func TestFailoverTransportSendsOtherRequestsDirectWhileProbing(t *testing.T) {
	probeStarted := make(chan struct{})
	releaseProbe := make(chan struct{})
	defer close(releaseProbe)
	var proxyCalls, directCalls atomic.Int32
	transport := &failoverTransport{
		proxy: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			if proxyCalls.Add(1) == 1 {
				close(probeStarted)
				<-releaseProbe
			}
			return jsonResponse(nil), nil
		}),
		direct: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			directCalls.Add(1)
			return jsonResponse(nil), nil
		}),
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		retryAt: time.Now().Add(-time.Second),
	}
	client := &http.Client{Transport: transport}
	probeDone := make(chan error, 1)
	go func() {
		response, err := client.Get("https://api.themoviedb.org/3/search/multi")
		if err == nil {
			response.Body.Close()
		}
		probeDone <- err
	}()
	select {
	case <-probeStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("VLESS probe did not start")
	}

	response, err := client.Get("https://api.themoviedb.org/3/search/multi")
	if err != nil {
		t.Fatalf("TMDB request during probe failed: %v", err)
	}
	response.Body.Close()
	if proxyCalls.Load() != 1 || directCalls.Load() != 1 {
		t.Fatalf("during probe: proxy calls = %d, direct calls = %d; want 1 and 1", proxyCalls.Load(), directCalls.Load())
	}
	releaseProbe <- struct{}{}
	if err := <-probeDone; err != nil {
		t.Fatalf("VLESS probe failed: %v", err)
	}
	response, err = client.Get("https://api.themoviedb.org/3/search/multi")
	if err != nil {
		t.Fatalf("TMDB request after probe failed: %v", err)
	}
	response.Body.Close()
	if proxyCalls.Load() != 2 || directCalls.Load() != 1 {
		t.Fatalf("after probe: proxy calls = %d, direct calls = %d; want 2 and 1", proxyCalls.Load(), directCalls.Load())
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
	if !errors.Is(err, context.Canceled) || directCalls != 0 || !transport.retryAt.IsZero() {
		t.Fatalf("error = %v, direct calls = %d, retry at = %v", err, directCalls, transport.retryAt)
	}
}
