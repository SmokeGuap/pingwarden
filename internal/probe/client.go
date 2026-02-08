package probe

import (
	"errors"
	"net"
	"net/http"
	"net/netip"
	"syscall"
	"time"
)

var ErrBlockedAddress = errors.New("target address is not allowed")

var extraBlocked = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("240.0.0.0/4"),
}

func IsBlocked(ip netip.Addr) bool {
	ip = ip.Unmap()
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}
	for _, p := range extraBlocked {
		if p.Contains(ip) {
			return true
		}
	}
	return false
}

func guard(_, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return ErrBlockedAddress
	}
	ip, err := netip.ParseAddr(host)
	if err != nil || IsBlocked(ip) {
		return ErrBlockedAddress
	}
	return nil
}

func NewClient(allowPrivate bool) *http.Client {
	d := &net.Dialer{Timeout: 10 * time.Second}
	if !allowPrivate {
		d.Control = guard
	}
	return &http.Client{
		Transport: &http.Transport{
			DialContext:         d.DialContext,
			TLSHandshakeTimeout: 10 * time.Second,
			IdleConnTimeout:     90 * time.Second,
		},
	}
}
