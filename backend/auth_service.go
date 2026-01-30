package cashtrack

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	apiv1 "cashtrack/backend/gen/api/v1"
	"cashtrack/backend/gen/api/v1/apiv1connect"
	"connectrpc.com/connect"
	"connectrpc.com/validate"
)

type AuthService struct {
	db *Db
}

type AuthServiceHandler Handler

func NewAuthServiceHandler(db *Db) *AuthServiceHandler {
	service := &AuthService{db: db}
	path, handler := apiv1connect.NewAuthServiceHandler(
		service,
		connect.WithInterceptors(validate.NewInterceptor(), NewAuthInterceptor(db)),
	)
	return &AuthServiceHandler{Path: path, Handler: handler}
}

func (s *AuthService) Me(ctx context.Context, req *apiv1.AuthMeRequest) (*apiv1.AuthMeResponse, error) {
	user, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}
	return &apiv1.AuthMeResponse{User: user}, nil
}

func (s *AuthService) Logout(ctx context.Context, req *apiv1.AuthLogoutRequest) (*apiv1.AuthLogoutResponse, error) {
	callInfo, ok := connect.CallInfoForHandlerContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.New("missing call info"))
	}

	sessionID, ok := sessionIDFromHeader(callInfo.RequestHeader())
	if ok {
		if sessionUUID, err := parseSessionID(sessionID); err == nil {
			_, _ = s.db.Queries.DeleteSession(ctx, sessionUUID)
		}
	}

	cookie := &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   isSecureRequest(callInfo.RequestHeader()),
	}
	callInfo.ResponseHeader().Add("Set-Cookie", cookie.String())

	return &apiv1.AuthLogoutResponse{}, nil
}

func isSecureRequest(header http.Header) bool {
	if strings.EqualFold(header.Get("X-Forwarded-Proto"), "https") {
		return true
	}
	forwarded := strings.ToLower(header.Get("Forwarded"))
	return strings.Contains(forwarded, "proto=https")
}
