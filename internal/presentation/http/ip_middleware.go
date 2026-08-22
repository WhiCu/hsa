package http

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/netip"
	"strings"
)

var ErrClientIPUnavailable = errors.New("http: client ip unavailable")

const forwardedForHeader = "X-Forwarded-For"

func NewClientIPMiddleware(
	log *slog.Logger,
	trustedProxies ...string,
) (func(http.Handler) http.Handler, error) {
	trusted, err := parseCIDRList(trustedProxies)
	if err != nil {
		return nil, err
	}

	return clientIPMiddleware(log, trusted), nil
}

func clientIPMiddleware(
	log *slog.Logger,
	trusted []netip.Prefix,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, ok := extractClientIP(r, trusted)
			if !ok {
				log.WarnContext(r.Context(), "failed to extract client ip from request",
					slog.String("remote_addr", r.RemoteAddr),
				)
			} else {
				log.DebugContext(r.Context(), "client ip resolved",
					slog.String("client_ip", ip.String()),
					slog.String("remote_addr", r.RemoteAddr),
				)
				r = r.WithContext(WithClientIP(r.Context(), ip))
			}

			next.ServeHTTP(w, r)
		})
	}
}
func extractClientIP(
	r *http.Request,
	trusted []netip.Prefix,
) (netip.Addr, bool) {
	remoteIP, ok := parseRemoteAddr(r.RemoteAddr)
	if !ok {
		return netip.Addr{}, false
	}

	if !isTrustedProxy(remoteIP, trusted) {
		return remoteIP, true
	}

	if ip, found := firstUntrustedFromRight(
		r.Header.Get(forwardedForHeader),
		trusted,
	); found {
		return ip, true
	}

	return remoteIP, true
}

func firstUntrustedFromRight(
	value string,
	trusted []netip.Prefix,
) (netip.Addr, bool) {
	// ⚡ Bolt: Iterate from right to left manually to avoid allocating strings.Split slice
	for i := len(value) - 1; i >= 0; {
		end := i
		commaIdx := strings.LastIndexByte(value[:i+1], ',')

		var part string
		if commaIdx == -1 {
			part = strings.TrimSpace(value[:end+1])
			i = -1
		} else {
			part = strings.TrimSpace(value[commaIdx+1 : end+1])
			i = commaIdx - 1
		}

		if part == "" {
			continue
		}

		ip, err := netip.ParseAddr(part)
		if err != nil {
			continue
		}

		if !isTrustedProxy(ip, trusted) {
			return ip, true
		}
	}

	return netip.Addr{}, false
}

func isTrustedProxy(ip netip.Addr, trusted []netip.Prefix) bool {
	for _, prefix := range trusted {
		if prefix.Contains(ip) {
			return true
		}
	}

	return false
}

func parseCIDRList(cidrs []string) ([]netip.Prefix, error) {
	out := make([]netip.Prefix, 0, len(cidrs))

	for _, cidr := range cidrs {
		prefix, err := netip.ParsePrefix(strings.TrimSpace(cidr))
		if err != nil {
			return nil, fmt.Errorf("parse trusted proxy CIDR %q: %w", cidr, err)
		}

		out = append(out, prefix.Masked())
	}

	return out, nil
}

func parseRemoteAddr(remoteAddr string) (netip.Addr, bool) {
	if addrPort, err := netip.ParseAddrPort(remoteAddr); err == nil {
		return addrPort.Addr(), true
	}

	if addr, err := netip.ParseAddr(remoteAddr); err == nil {
		return addr, true
	}

	return netip.Addr{}, false
}
