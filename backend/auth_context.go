package cashtrack

import (
	"context"
	"net/http"
	"time"

	apiv1 "cashtrack/backend/gen/api/v1"
	"connectrpc.com/connect"
)

type authUserContextKey struct{}

func contextWithUser(ctx context.Context, user *apiv1.User) context.Context {
	return context.WithValue(ctx, authUserContextKey{}, user)
}

func userFromContext(ctx context.Context) (*apiv1.User, bool) {
	user, ok := ctx.Value(authUserContextKey{}).(*apiv1.User)
	return user, ok
}

func NewAuthInterceptor(db *Db) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			user, ok := userFromRequest(ctx, db, req.Header())
			if ok {
				ctx = contextWithUser(ctx, user)
			}
			return next(ctx, req)
		}
	}
}

func userFromRequest(ctx context.Context, db *Db, header http.Header) (*apiv1.User, bool) {
	sessionID, ok := sessionIDFromHeader(header)
	if !ok {
		return nil, false
	}

	user, expiresAt, err := getUserBySession(ctx, db, sessionID)
	if err != nil {
		return nil, false
	}
	if time.Now().After(expiresAt) {
		return nil, false
	}
	return user, true
}

func sessionIDFromHeader(header http.Header) (string, bool) {
	cookieHeader := header.Get("Cookie")
	if cookieHeader == "" {
		return "", false
	}

	req := http.Request{Header: http.Header{"Cookie": []string{cookieHeader}}}
	cookie, err := req.Cookie(sessionCookieName)
	if err != nil || cookie.Value == "" {
		return "", false
	}
	return cookie.Value, true
}
