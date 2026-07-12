package parser

import (
	"fmt"
	"net/url"
	"strings"
)

// VlessNode ساختار اطلاعات استخراج شده از یک لینک VLESS را نگه می‌دارد
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

// ParseVless یک لینک خام را می‌گیرد و آن را به آبجکت VlessNode تبدیل می‌کند
func ParseVless(rawLink string) (*VlessNode, error) {
	if !strings.HasPrefix(rawLink, "vless://") {
		return nil, fmt.Errorf("unsupported or invalid protocol, only vless:// is supported")
	}

	// فرمت استاندارد: vless://uuid@address:port?query_params#name
	parsedUrl, err := url.Parse(rawLink)
	if err != nil {
		return nil, fmt.Errorf("failed to parse link: %v", err)
	}

	uuid := parsedUrl.User.Username()
	address := parsedUrl.Hostname()
	port := parsedUrl.Port()
	name := parsedUrl.Fragment

	// اگر اسم در فرمت URL انکود شده بود (مثل خط تیره‌ها یا فاصله‌ها)
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
		Network:    query.Get("type"), // نوع شبکه مثل ws, tcp, grpc
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
