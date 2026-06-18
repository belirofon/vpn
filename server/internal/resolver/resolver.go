package resolver

import (
	"context"
	"net"
	"time"
)

func ResolveIP(host string, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return "", err
	}

	// Prefer IPv4
	for _, ip := range ips {
		if ipv4 := ip.IP.To4(); ipv4 != nil {
			return ipv4.String(), nil
		}
	}

	if len(ips) > 0 {
		return ips[0].IP.String(), nil
	}

	return "", &net.DNSError{Err: "no addresses found", Name: host, Server: "", IsTimeout: false, IsTemporary: false, IsNotFound: true}
}
