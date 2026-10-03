package config

import "testing"

func TestLoadReadsBackupVLESSURL(t *testing.T) {
	t.Setenv("BACKUP_VLESS_URL", "  vless://example  ")
	if got := Load().VLESSURL; got != "vless://example" {
		t.Fatalf("VLESS URL = %q; want vless://example", got)
	}
}
