package resolver

import (
	"context"
	"net"
	"time"
)

// ResolveIP resolves a host to an IPv4 address using the given context.
func ResolveIP(ctx context.Context, host string, timeout time.Duration) (string, error) {
	resolveCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ips, err := net.DefaultResolver.LookupIPAddr(resolveCtx, host)
	if err != nil {
		return "", err
	}

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
