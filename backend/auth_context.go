package cashtrack

import (
	"context"
	"net/http"
	"time"

	"connectrpc.com/connect"
)

type AuthUser struct {
	ID       int32  `json:"id"`
	Username string `json:"username"`
}

type authUserContextKey struct{}

func contextWithUser(ctx context.Context, user AuthUser) context.Context {
	return context.WithValue(ctx, authUserContextKey{}, user)
}

func userFromContext(ctx context.Context) (AuthUser, bool) {
	user, ok := ctx.Value(authUserContextKey{}).(AuthUser)
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

func userFromRequest(ctx context.Context, db *Db, header http.Header) (AuthUser, bool) {
	cookieHeader := header.Get("Cookie")
	if cookieHeader == "" {
		return AuthUser{}, false
	}

	req := http.Request{Header: http.Header{"Cookie": []string{cookieHeader}}}
	cookie, err := req.Cookie(sessionCookieName)
	if err != nil || cookie.Value == "" {
		return AuthUser{}, false
	}

	user, expiresAt, err := getUserBySession(ctx, db, cookie.Value)
	if err != nil {
		return AuthUser{}, false
	}
	if time.Now().After(expiresAt) {
		return AuthUser{}, false
	}
	return user, true
}
