package entity

import "testing"

func validAllSetting() *AllSetting {
	return &AllSetting{
		WebPort:      2053,
		WebBasePath:  "/",
		SubPort:      2096,
		SubPath:      "/sub/",
		SubJsonPath:  "/json/",
		SubClashPath: "/clash/",
		TimeLocation: "Local",
	}
}

func TestAllSettingCheckValidAcceptsIPv6SubscriptionAddress(t *testing.T) {
	setting := validAllSetting()
	setting.SubIPv6Address = "2001:db8::1"

	if err := setting.CheckValid(); err != nil {
		t.Fatalf("expected IPv6 subscription address to be valid: %v", err)
	}
}

func TestAllSettingCheckValidRejectsIPv4SubscriptionAddress(t *testing.T) {
	setting := validAllSetting()
	setting.SubIPv6Address = "192.0.2.1"

	if err := setting.CheckValid(); err == nil {
		t.Fatal("expected IPv4 subscription address to be rejected")
	}
}
