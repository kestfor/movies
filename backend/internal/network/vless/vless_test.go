package vless

import (
	"encoding/json"
	"strings"
	"testing"

	xraycore "github.com/xtls/xray-core/core"
)

const testVLESSURL = "vless://11111111-1111-4111-8111-111111111111@203.0.113.10:443?mode=auto&path=%2Ftelegram&security=reality&encryption=none&extra=%7BscMaxEachPostBytes%3D1000000%2C%20xPaddingBytes%3D100-1000%7D&pbk=AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA&host=example-host&fp=chrome&spx=%2Fprobe&type=xhttp&sni=example.com&sid=0011223344556677#test"
const testGRPCVLESSURL = "vless://11111111-1111-4111-8111-111111111111@203.0.113.10:443?type=grpc&serviceName=telegram&mode=multi&authority=grpc.example.com&security=reality&encryption=none&pbk=AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA&fp=chrome&sni=example.com&sid=0011223344556677#test"

func TestParseVLESSURL(t *testing.T) {
	t.Parallel()

	config, err := parseVLESSURL(testVLESSURL)
	if err != nil {
		t.Fatalf("parseVLESSURL returned error: %v", err)
	}

	if config.Address != "203.0.113.10" || config.Port != 443 {
		t.Fatalf("unexpected server: %s:%d", config.Address, config.Port)
	}
	if config.ID != "11111111-1111-4111-8111-111111111111" {
		t.Fatalf("unexpected ID: %q", config.ID)
	}
	if config.Transport != "xhttp" || config.Mode != "auto" || config.Path != "/telegram" || config.Host != "example-host" {
		t.Fatalf("unexpected XHTTP config: %+v", config)
	}
	if config.ServerName != "example.com" || config.Fingerprint != "chrome" || config.SpiderX != "/probe" {
		t.Fatalf("unexpected REALITY config: %+v", config)
	}
	if got := config.Extra["scMaxEachPostBytes"]; got != int64(1_000_000) {
		t.Fatalf("unexpected scMaxEachPostBytes: %#v", got)
	}
	if got := config.Extra["xPaddingBytes"]; got != "100-1000" {
		t.Fatalf("unexpected xPaddingBytes: %#v", got)
	}
}

func TestParseGRPCVLESSURL(t *testing.T) {
	t.Parallel()

	config, err := parseVLESSURL(testGRPCVLESSURL)
	if err != nil {
		t.Fatalf("parseVLESSURL returned error: %v", err)
	}
	if config.Transport != "grpc" || config.ServiceName != "telegram" || config.Authority != "grpc.example.com" || config.Mode != "multi" {
		t.Fatalf("unexpected gRPC config: %+v", config)
	}
	config, err = parseVLESSURL(strings.Replace(testGRPCVLESSURL, "&mode=multi", "", 1))
	if err != nil || config.Mode != "gun" {
		t.Fatalf("default gRPC mode = %q, error = %v; want gun", config.Mode, err)
	}
}

func TestParseVLESSURLRejectsUnsupportedProfiles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		url  string
	}{
		{name: "wrong scheme", url: "https://example.com"},
		{name: "missing port", url: strings.Replace(testVLESSURL, ":443", "", 1)},
		{name: "wrong transport", url: strings.Replace(testVLESSURL, "type=xhttp", "type=ws", 1)},
		{name: "missing gRPC service name", url: strings.Replace(testGRPCVLESSURL, "serviceName=telegram&", "", 1)},
		{name: "wrong gRPC mode", url: strings.Replace(testGRPCVLESSURL, "mode=multi", "mode=invalid", 1)},
		{name: "wrong security", url: strings.Replace(testVLESSURL, "security=reality", "security=tls", 1)},
		{name: "invalid public key", url: strings.Replace(testVLESSURL, "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", "invalid", 1)},
		{name: "invalid short ID", url: strings.Replace(testVLESSURL, "0011223344556677", "xyz", 1)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if _, err := parseVLESSURL(test.url); err == nil {
				t.Fatal("expected parseVLESSURL to return an error")
			}
		})
	}
}

func TestVLESSConfigStartsXray(t *testing.T) {
	for _, test := range []struct {
		name string
		url  string
	}{
		{name: "XHTTP", url: testVLESSURL},
		{name: "gRPC", url: testGRPCVLESSURL},
	} {
		t.Run(test.name, func(t *testing.T) {
			config, err := parseVLESSURL(test.url)
			if err != nil {
				t.Fatalf("parseVLESSURL returned error: %v", err)
			}
			rawConfig, err := config.xrayConfig()
			if err != nil {
				t.Fatalf("xrayConfig returned error: %v", err)
			}
			var parsed struct {
				Outbounds []struct {
					StreamSettings struct {
						Network       string          `json:"network"`
						XHTTPSettings json.RawMessage `json:"xhttpSettings"`
						GRPCSettings  struct {
							ServiceName string `json:"serviceName"`
							Authority   string `json:"authority"`
							MultiMode   bool   `json:"multiMode"`
						} `json:"grpcSettings"`
					} `json:"streamSettings"`
				} `json:"outbounds"`
			}
			if err := json.Unmarshal(rawConfig, &parsed); err != nil {
				t.Fatalf("decode Xray config: %v", err)
			}
			if len(parsed.Outbounds) != 1 || parsed.Outbounds[0].StreamSettings.Network != config.Transport {
				t.Fatalf("unexpected Xray network: %s", rawConfig)
			}
			settings := parsed.Outbounds[0].StreamSettings
			if config.Transport == "grpc" {
				if settings.GRPCSettings.ServiceName != "telegram" || settings.GRPCSettings.Authority != "grpc.example.com" || !settings.GRPCSettings.MultiMode || len(settings.XHTTPSettings) != 0 {
					t.Fatalf("unexpected gRPC settings: %s", rawConfig)
				}
			} else if len(settings.XHTTPSettings) == 0 || settings.GRPCSettings.ServiceName != "" {
				t.Fatalf("unexpected XHTTP settings: %s", rawConfig)
			}
			instance, err := xraycore.StartInstance("json", rawConfig)
			if err != nil {
				t.Fatalf("StartInstance returned error: %v", err)
			}
			if err := instance.Close(); err != nil {
				t.Fatalf("close Xray instance: %v", err)
			}
		})
	}
}
