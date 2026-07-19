package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Mhmdrz-rasekh/xray-cli/parser"
)

type XrayConfig struct {
	Log       LogConfig        `json:"log"`
	API       *ApiConfig       `json:"api,omitempty"`
	Stats     *StatsConfig     `json:"stats,omitempty"`
	Policy    *PolicyConfig    `json:"policy,omitempty"`
	DNS       *DNSConfig       `json:"dns,omitempty"`
	FakeDNS   []FakeDNSConfig  `json:"fakedns,omitempty"`
	Inbounds  []InboundConfig  `json:"inbounds"`
	Outbounds []OutboundConfig `json:"outbounds"`
	Routing   RoutingConfig    `json:"routing,omitempty"`
}

type ApiConfig struct {
	Tag      string   `json:"tag"`
	Services []string `json:"services"`
}

type StatsConfig struct{}

type PolicyConfig struct {
	System map[string]interface{} `json:"system"`
}

type DNSConfig struct {
	Servers []interface{} `json:"servers"`
	QueryStrategy string        `json:"queryStrategy,omitempty"`
}

type FakeDNSConfig struct {
	IpPool   string `json:"ipPool"`
	PoolSize int    `json:"poolSize"`
}

type LogConfig struct {
	LogLevel string `json:"loglevel"`
}

type InboundConfig struct {
	Listen   string                 `json:"listen,omitempty"`
	Tag      string                 `json:"tag,omitempty"`
	Port     int                    `json:"port,omitempty"`
	Protocol string                 `json:"protocol"`
	Settings map[string]interface{} `json:"settings"`
	Sniffing map[string]interface{} `json:"sniffing,omitempty"`
}

type OutboundConfig struct {
	Protocol       string                 `json:"protocol"`
	Settings       map[string]interface{} `json:"settings"`
	StreamSettings map[string]interface{} `json:"streamSettings,omitempty"`
	Tag            string                 `json:"tag"`
}

type RoutingConfig struct {
	DomainStrategy string                   `json:"domainStrategy,omitempty"`
	Rules          []map[string]interface{} `json:"rules,omitempty"`
}

func GenerateConfig(node *parser.VlessNode, mode string, socksPort int) (string, error) {
	var inbounds []InboundConfig
    var fakeDNS []FakeDNSConfig
    outbounds := []OutboundConfig{}
    rules := []map[string]interface{}{}

    // Global Safe DNS - Forces IPv4 to prevent remote IPv6 sinkholes
    dnsConfig := &DNSConfig{
        Servers:       []interface{}{"8.8.8.8", "1.1.1.1", "localhost"},
        QueryStrategy: "UseIPv4",
    }

	portNum, err := strconv.Atoi(node.Port)
	if err != nil || portNum == 0 { portNum = 443 }

	network := node.Network
	if network == "" { network = "tcp" }
	security := node.Security
	if security == "" { security = "none" }

	targetSNI := node.SNI
	if targetSNI == "" {
		if node.Host != "" { targetSNI = node.Host } else { targetSNI = node.Address }
	}

	streamSettings := map[string]interface{}{"network": network, "security": security}

	if security == "reality" {
		rs := map[string]interface{}{"show": false, "fingerprint": node.FP, "serverName": targetSNI, "publicKey": node.PBK, "shortId": node.SID}
		if node.FP == "" { rs["fingerprint"] = "chrome" }
		if node.SpiderX != "" { rs["spiderX"] = node.SpiderX }
		streamSettings["realitySettings"] = rs
	} else if security == "tls" {
		ts := map[string]interface{}{"serverName": targetSNI, "fingerprint": node.FP}
		if node.ALPN != "" { ts["alpn"] = strings.Split(node.ALPN, ",") }
		streamSettings["tlsSettings"] = ts
	}

	if network == "ws" {
		ws := map[string]interface{}{"path": node.Path}
		if node.Host != "" { ws["headers"] = map[string]string{"Host": node.Host} }
		streamSettings["wsSettings"] = ws
	}

	enc := node.Encryption
	if enc == "" { enc = "none" }
	userObj := map[string]interface{}{"id": node.UUID, "encryption": enc, "level": 0}
	if node.Flow != "" { userObj["flow"] = node.Flow }

	outbounds = append(outbounds, OutboundConfig{
		Protocol: "vless",
		Tag:      "proxy",
		Settings: map[string]interface{}{
			"vnext": []map[string]interface{}{{"address": node.Address, "port": portNum, "users": []map[string]interface{}{userObj}}},
		},
		StreamSettings: streamSettings,
	})
	outbounds = append(outbounds, OutboundConfig{Protocol: "freedom", Tag: "direct", Settings: map[string]interface{}{}})
	outbounds = append(outbounds, OutboundConfig{Protocol: "blackhole", Tag: "block", Settings: map[string]interface{}{}})

	if mode == "tun" {
			fakeDNS = append(fakeDNS, FakeDNSConfig{IpPool: "198.18.0.0/15", PoolSize: 65535})
			dnsConfig.Servers = []interface{}{"fakedns", "8.8.8.8"} // Override for TUN
			inbounds = append(inbounds, InboundConfig{
				Protocol: "tun", Tag: "tun-in",
				// INJECTED fd00::/126 to capture and proxy IPv6 traffic
				Settings: map[string]interface{}{"network": "10.0.0.1/30,fd00::/126", "system": true, "autoRoute": true, "strictRoute": false},
				Sniffing: map[string]interface{}{"enabled": true, "destOverride": []string{"http", "tls", "quic", "fakedns"}},
			})
			outbounds = append(outbounds, OutboundConfig{Protocol: "dns", Tag: "dns-out", Settings: map[string]interface{}{}})
			rules = append(rules, map[string]interface{}{"type": "field", "inboundTag": []string{"tun-in"}, "port": 53, "network": "udp", "outboundTag": "dns-out"})
		} else {
			httpPort := socksPort + 1
			sniff := map[string]interface{}{"enabled": true, "destOverride": []string{"http", "tls", "quic"}}
			inbounds = append(inbounds,
				InboundConfig{Port: socksPort, Protocol: "socks", Settings: map[string]interface{}{"auth": "noauth", "udp": true}, Sniffing: sniff},
				InboundConfig{Port: httpPort, Protocol: "http", Settings: map[string]interface{}{}, Sniffing: sniff},
			)
		}

	inbounds = append(inbounds, InboundConfig{
		Listen: "127.0.0.1", Port: 10085, Protocol: "dokodemo-door", Tag: "api",
		Settings: map[string]interface{}{"address": "127.0.0.1"},
	})
	rules = append(rules, map[string]interface{}{"type": "field", "inboundTag": []string{"api"}, "outboundTag": "api"})

	rules = append(rules, map[string]interface{}{"type": "field", "ip": []string{"geoip:private"}, "outboundTag": "direct"})

    // Force browser to abandon QUIC and fallback to standard TCP
    rules = append(rules, map[string]interface{}{
        "type": "field",
        "port": 443,
        "network": "udp",
        "outboundTag": "block",
    })

	cfg := XrayConfig{
		Log: LogConfig{LogLevel: "warning"},
		API: &ApiConfig{Tag: "api", Services: []string{"StatsService"}},
		Stats: &StatsConfig{},
		Policy: &PolicyConfig{
			System: map[string]interface{}{
				"statsOutboundUplink": true, "statsOutboundDownlink": true,
			},
		},
		DNS: dnsConfig, FakeDNS: fakeDNS, Inbounds: inbounds, Outbounds: outbounds,
		Routing: RoutingConfig{DomainStrategy: "IPIfNonMatch", Rules: rules},
	}

	configDir, _ := os.UserConfigDir()
	appDir := filepath.Join(configDir, "xray-cli")
	outputPath := filepath.Join(appDir, "xray_run_config.json")
	fileData, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil { return "", err }
	os.WriteFile(outputPath, fileData, 0644)
	return outputPath, nil
}
