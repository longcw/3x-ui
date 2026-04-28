package service

import (
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/mhsanaei/3x-ui/v2/database"
	"github.com/mhsanaei/3x-ui/v2/database/model"
)

func mustIPAddr(t *testing.T, address string) net.Addr {
	t.Helper()
	ip := net.ParseIP(address)
	if ip == nil {
		t.Fatalf("parse ip %q", address)
	}
	return &net.IPAddr{IP: ip}
}

func initSettingTestDB(t *testing.T) {
	t.Helper()
	binDir := t.TempDir()
	t.Setenv("XUI_BIN_FOLDER", binDir)
	if err := os.WriteFile(filepath.Join(binDir, "config.json"), []byte(`{"log":{"access":"none"}}`), 0o600); err != nil {
		t.Fatalf("write xray config: %v", err)
	}
	if err := database.InitDB(filepath.Join(t.TempDir(), "3x-ui.db")); err != nil {
		t.Fatalf("init test database: %v", err)
	}
}

func TestSelectPublicIPv6AddressPrefersGlobalNonPrivateIPv6(t *testing.T) {
	got := selectPublicIPv6Address([]net.Addr{
		mustIPAddr(t, "127.0.0.1"),
		mustIPAddr(t, "::1"),
		mustIPAddr(t, "fe80::1"),
		mustIPAddr(t, "fd00::1"),
		mustIPAddr(t, "2001:4860::123"),
	})

	if got != "2001:4860::123" {
		t.Fatalf("expected public IPv6 address, got %q", got)
	}
}

func TestGetDefaultSettingsPrefillsDetectedIPv6WhenUnset(t *testing.T) {
	initSettingTestDB(t)
	orig := detectPublicIPv6Address
	detectPublicIPv6Address = func() string { return "2001:4860::123" }
	t.Cleanup(func() { detectPublicIPv6Address = orig })

	result, err := (&SettingService{}).GetDefaultSettings("example.com")
	if err != nil {
		t.Fatalf("get default settings: %v", err)
	}

	settings := result.(map[string]any)
	if settings["subIPv6Address"] != "2001:4860::123" {
		t.Fatalf("expected detected IPv6 default, got %#v", settings["subIPv6Address"])
	}
}

func TestGetDefaultSettingsKeepsConfiguredIPv6(t *testing.T) {
	initSettingTestDB(t)
	if err := database.GetDB().Create(&model.Setting{Key: "subIPv6Address", Value: "2001:4860::456"}).Error; err != nil {
		t.Fatalf("seed IPv6 setting: %v", err)
	}
	orig := detectPublicIPv6Address
	detectPublicIPv6Address = func() string { return "2001:4860::123" }
	t.Cleanup(func() { detectPublicIPv6Address = orig })

	result, err := (&SettingService{}).GetDefaultSettings("example.com")
	if err != nil {
		t.Fatalf("get default settings: %v", err)
	}

	settings := result.(map[string]any)
	if settings["subIPv6Address"] != "2001:4860::456" {
		t.Fatalf("expected configured IPv6 to be preserved, got %#v", settings["subIPv6Address"])
	}
}
