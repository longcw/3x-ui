package sub

import (
	"encoding/base64"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/goccy/go-json"
	"github.com/mhsanaei/3x-ui/v2/database"
	"github.com/mhsanaei/3x-ui/v2/database/model"
)

func subServiceWithIPv6Setting(t *testing.T) *SubService {
	t.Helper()
	if err := database.InitDB(filepath.Join(t.TempDir(), "3x-ui.db")); err != nil {
		t.Fatalf("init test database: %v", err)
	}
	if err := database.GetDB().Create(&model.Setting{Key: "subIPv6Address", Value: "2001:db8::1"}).Error; err != nil {
		t.Fatalf("seed IPv6 setting: %v", err)
	}
	return &SubService{remarkModel: "-ieo"}
}

func TestAppendIPv6EndpointsAddsBareAndURIFormattedAddress(t *testing.T) {
	endpoints := []subscriptionEndpoint{{
		Address:    "example.com",
		URIAddress: "example.com",
		Port:       443,
		Remark:     "",
		ForceTLS:   "same",
	}}

	got := appendIPv6Endpoints(endpoints, "2001:db8::1")

	if len(got) != 2 {
		t.Fatalf("expected primary and IPv6 endpoints, got %d", len(got))
	}
	if got[1].Address != "2001:db8::1" {
		t.Fatalf("expected bare IPv6 address for non-URI consumers, got %q", got[1].Address)
	}
	if got[1].URIAddress != "[2001:db8::1]" {
		t.Fatalf("expected bracketed IPv6 address for URI consumers, got %q", got[1].URIAddress)
	}
	if got[1].Port != 443 {
		t.Fatalf("expected IPv6 endpoint to reuse the primary port, got %d", got[1].Port)
	}
	if got[1].Remark != "IPv6" {
		t.Fatalf("expected IPv6 endpoint remark suffix, got %q", got[1].Remark)
	}
	if got[1].ForceTLS != "same" {
		t.Fatalf("expected IPv6 endpoint to preserve forceTls, got %q", got[1].ForceTLS)
	}
}

func TestAppendIPv6EndpointsDoesNotDuplicateExistingIPv6Address(t *testing.T) {
	endpoints := []subscriptionEndpoint{{
		Address:    "2001:db8::1",
		URIAddress: "[2001:db8::1]",
		Port:       443,
		Remark:     "",
		ForceTLS:   "same",
	}}

	got := appendIPv6Endpoints(endpoints, "[2001:db8::1]")

	if len(got) != 1 {
		t.Fatalf("expected existing IPv6 endpoint not to be duplicated, got %d endpoints", len(got))
	}
}

func TestBuildEndpointURLLinksBracketsIPv6URIHost(t *testing.T) {
	service := &SubService{remarkModel: "-ieo"}
	endpoints := appendIPv6Endpoints([]subscriptionEndpoint{{
		Address:    "example.com",
		URIAddress: "example.com",
		Port:       443,
		Remark:     "",
		ForceTLS:   "same",
	}}, "2001:db8::1")

	links := service.buildEndpointURLLinks(
		endpoints,
		map[string]string{"type": "tcp", "security": "none"},
		"none",
		func(dest string, port int) string {
			return "vless://00000000-0000-0000-0000-000000000000@" + dest + ":" + strconv.Itoa(port)
		},
		func(endpoint subscriptionEndpoint) string {
			return service.genRemark(&model.Inbound{Remark: "node"}, "client@example.com", endpoint.Remark)
		},
	)

	lines := strings.Split(links, "\n")
	if len(lines) != 2 {
		t.Fatalf("expected IPv4/domain and IPv6 links, got %d", len(lines))
	}
	if !strings.Contains(lines[1], "@[2001:db8::1]:443") {
		t.Fatalf("expected IPv6 URI host to be bracketed, got %q", lines[1])
	}
	if !strings.Contains(lines[1], "IPv6") {
		t.Fatalf("expected IPv6 link remark to be unique, got %q", lines[1])
	}
}

func TestBuildVmessEndpointLinksKeepsBareIPv6Address(t *testing.T) {
	service := &SubService{remarkModel: "-ieo"}
	endpoints := appendIPv6Endpoints([]subscriptionEndpoint{{
		Address:    "example.com",
		URIAddress: "example.com",
		Port:       443,
		Remark:     "",
		ForceTLS:   "same",
	}}, "2001:db8::1")

	links := service.buildVmessEndpointLinks(
		endpoints,
		map[string]any{"v": "2", "type": "none", "tls": "none"},
		&model.Inbound{Remark: "node"},
		"client@example.com",
	)

	lines := strings.Split(links, "\n")
	if len(lines) != 2 {
		t.Fatalf("expected IPv4/domain and IPv6 vmess links, got %d", len(lines))
	}
	payload, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(lines[1], "vmess://"))
	if err != nil {
		t.Fatalf("decode vmess payload: %v", err)
	}
	var obj map[string]any
	if err := json.Unmarshal(payload, &obj); err != nil {
		t.Fatalf("unmarshal vmess payload: %v", err)
	}
	if obj["add"] != "2001:db8::1" {
		t.Fatalf("expected bare IPv6 vmess add field, got %#v", obj["add"])
	}
	if !strings.Contains(obj["ps"].(string), "IPv6") {
		t.Fatalf("expected unique IPv6 vmess remark, got %q", obj["ps"])
	}
}

func TestSubJsonConfigDuplicatesBareIPv6Address(t *testing.T) {
	subService := subServiceWithIPv6Setting(t)
	jsonService := NewSubJsonService("", "", "", "", subService)
	inbound := &model.Inbound{
		Port:           443,
		Protocol:       model.VLESS,
		Remark:         "node",
		Settings:       `{"encryption":"none"}`,
		StreamSettings: `{"network":"tcp","security":"none","tcpSettings":{"header":{"type":"none"}}}`,
	}
	client := model.Client{ID: "00000000-0000-0000-0000-000000000000", Email: "client@example.com"}

	configs := jsonService.getConfig(inbound, client, "example.com")

	if len(configs) != 2 {
		t.Fatalf("expected JSON configs for primary and IPv6 endpoints, got %d", len(configs))
	}
	if got := vlessJSONAddress(t, configs[0]); got != "example.com" {
		t.Fatalf("expected primary JSON address example.com, got %q", got)
	}
	if got := vlessJSONAddress(t, configs[1]); got != "2001:db8::1" {
		t.Fatalf("expected bare IPv6 JSON address, got %q", got)
	}
	if !strings.Contains(jsonRemark(t, configs[1]), "IPv6") {
		t.Fatalf("expected IPv6 JSON remark to be unique, got %q", jsonRemark(t, configs[1]))
	}
}

func TestSubClashProxiesDuplicateBareIPv6Server(t *testing.T) {
	subService := subServiceWithIPv6Setting(t)
	clashService := NewSubClashService(subService)
	inbound := &model.Inbound{
		Port:           443,
		Protocol:       model.VLESS,
		Remark:         "node",
		Settings:       `{"encryption":"none"}`,
		StreamSettings: `{"network":"tcp","security":"none","tcpSettings":{"header":{"type":"none"}}}`,
	}
	client := model.Client{ID: "00000000-0000-0000-0000-000000000000", Email: "client@example.com"}

	proxies := clashService.getProxies(inbound, client, "example.com")

	if len(proxies) != 2 {
		t.Fatalf("expected Clash proxies for primary and IPv6 endpoints, got %d", len(proxies))
	}
	if got := proxies[0]["server"]; got != "example.com" {
		t.Fatalf("expected primary Clash server example.com, got %#v", got)
	}
	if got := proxies[1]["server"]; got != "2001:db8::1" {
		t.Fatalf("expected bare IPv6 Clash server, got %#v", got)
	}
	if !strings.Contains(proxies[1]["name"].(string), "IPv6") {
		t.Fatalf("expected IPv6 Clash proxy name to be unique, got %q", proxies[1]["name"])
	}
}

func vlessJSONAddress(t *testing.T, raw []byte) string {
	t.Helper()
	var config map[string]any
	if err := json.Unmarshal(raw, &config); err != nil {
		t.Fatalf("unmarshal JSON config: %v", err)
	}
	outbounds := config["outbounds"].([]any)
	outbound := outbounds[0].(map[string]any)
	settings := outbound["settings"].(map[string]any)
	vnext := settings["vnext"].([]any)
	server := vnext[0].(map[string]any)
	return server["address"].(string)
}

func jsonRemark(t *testing.T, raw []byte) string {
	t.Helper()
	var config map[string]any
	if err := json.Unmarshal(raw, &config); err != nil {
		t.Fatalf("unmarshal JSON config: %v", err)
	}
	return config["remarks"].(string)
}
