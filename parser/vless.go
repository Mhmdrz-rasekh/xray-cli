package parser

import (
	"fmt"
	"net/url"
	"strings"
)

type VlessNode struct {
	Name       string
	Address    string
	Port       string
	UUID       string
	Encryption string
	Security   string
	Network    string
	SNI        string
	FP         string
	PBK        string
	SID        string
	SpiderX    string
	ALPN       string
	Path       string
	Host       string
	Flow       string
}

func ParseVless(rawLink string) (*VlessNode, error) {
	if !strings.HasPrefix(rawLink, "vless://") {
		return nil, fmt.Errorf("unsupported or invalid protocol, only vless:// is supported")
	}

	parsedUrl, err := url.Parse(rawLink)
	if err != nil {
		return nil, fmt.Errorf("failed to parse link: %v", err)
	}

	uuid := parsedUrl.User.Username()
	address := parsedUrl.Hostname()
	port := parsedUrl.Port()
	name := parsedUrl.Fragment

	
	if decodedName, err := url.QueryUnescape(name); err == nil {
		name = decodedName
	}

	query := parsedUrl.Query()

	return &VlessNode{
		Name:       name,
		Address:    address,
		Port:       port,
		UUID:       uuid,
		Encryption: query.Get("encryption"),
		Security:   query.Get("security"),
		Network:    query.Get("type"), 
		SNI:        query.Get("sni"),
		FP:         query.Get("fp"),
		PBK:        query.Get("pbk"),
		SID:        query.Get("sid"),
		SpiderX:    query.Get("spx"),
		ALPN:       query.Get("alpn"),
		Path:       query.Get("path"),
		Host:       query.Get("host"),
		Flow:       query.Get("flow"),
	}, nil
}
