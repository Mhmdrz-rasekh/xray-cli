package core

import (
	"context"
	"net"
	"time"

	"github.com/Mhmdrz-rasekh/xray-cli/parser"
)


func MeasureHttpPing(node *parser.VlessNode) (time.Duration, error) {
	port := node.Port
	if port == "" {
		port = "443"
	}

	target := net.JoinHostPort(node.Address, port)


	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	start := time.Now()

	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", target)
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	duration := time.Since(start)
	return duration, nil
}
