package http

import (
	"context"
	"net/netip"

	"github.com/whicu/hsa/internal/domain/user"
)

type ctxKey struct{}

var (
	userIDContextKey = ctxKey{}
	clientIPKey      = ctxKey{}
)

func WithUserID(ctx context.Context, id user.UserID) context.Context {
	return context.WithValue(ctx, userIDContextKey, id)
}
func userIDFromContext(ctx context.Context) (user.UserID, bool) {
	id, ok := ctx.Value(userIDContextKey).(user.UserID)
	if !ok {
		return user.UserID{}, false
	}
	return id, true
}

func WithClientIP(ctx context.Context, ip netip.Addr) context.Context {
	return context.WithValue(ctx, clientIPKey, ip)
}

func ClientIPFromContext(ctx context.Context) (netip.Addr, bool) {
	ip, ok := ctx.Value(clientIPKey).(netip.Addr)
	return ip, ok
}
