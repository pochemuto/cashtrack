package cashtrack

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	db "cashtrack/backend/gen/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type AuthHandler Handler
type AuthMeHandler Handler
type AuthLogoutHandler Handler

const sessionCookieName = "session_id"
const sessionDuration = 7 * 24 * time.Hour

type idTokenClaims struct {
	Sub   string `json:"sub"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

func NewAuthHandler(db *Db) *AuthHandler {
	return &AuthHandler{
		Path: "/auth",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Info().
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Msg("Auth handler called")

			credential := r.URL.Query().Get("credential")
			if credential == "" {
				http.Error(w, "missing credential", http.StatusBadRequest)
				return
			}

			claims, err := parseIDToken(credential)
			if err != nil {
				http.Error(w, "invalid credential", http.StatusBadRequest)
				return
			}

			username := claims.Email
			if username == "" {
				username = claims.Sub
			}
			if username == "" {
				http.Error(w, "missing user info", http.StatusBadRequest)
				return
			}

			user, err := ensureUser(r.Context(), db, username)
			if err != nil {
				http.Error(w, "failed to create user", http.StatusInternalServerError)
				return
			}

			sessionID, expiresAt, err := createSession(r.Context(), db, user.ID)
			if err != nil {
				http.Error(w, "failed to create session", http.StatusInternalServerError)
				return
			}

			http.SetCookie(w, &http.Cookie{
				Name:     sessionCookieName,
				Value:    sessionID,
				Path:     "/",
				Expires:  expiresAt,
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
				Secure:   r.TLS != nil,
			})

			redirectURL := r.URL.Query().Get("redirect")
			if redirectURL == "" {
				redirectURL = "/"
			}
			http.Redirect(w, r, redirectURL, http.StatusFound)
		}),
	}
}

func NewAuthMeHandler(db *Db) *AuthMeHandler {
	return &AuthMeHandler{
		Path: "/auth/me",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(sessionCookieName)
			if err != nil || cookie.Value == "" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			user, expiresAt, err := getUserBySession(r.Context(), db, cookie.Value)
			if err != nil {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			if time.Now().After(expiresAt) {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(user); err != nil {
				http.Error(w, "failed to encode response", http.StatusInternalServerError)
			}
		}),
	}
}

func NewAuthLogoutHandler(db *Db) *AuthLogoutHandler {
	return &AuthLogoutHandler{
		Path: "/auth/logout",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}

			cookie, err := r.Cookie(sessionCookieName)
			if err == nil && cookie.Value != "" {
				_, _ = db.Queries.DeleteSession(r.Context(), cookie.Value)
			}

			http.SetCookie(w, &http.Cookie{
				Name:     sessionCookieName,
				Value:    "",
				Path:     "/",
				Expires:  time.Unix(0, 0),
				MaxAge:   -1,
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
				Secure:   r.TLS != nil,
			})

			w.WriteHeader(http.StatusNoContent)
		}),
	}
}

func parseIDToken(credential string) (idTokenClaims, error) {
	parts := strings.Split(credential, ".")
	if len(parts) < 2 {
		return idTokenClaims{}, errors.New("invalid token")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return idTokenClaims{}, err
	}
	var claims idTokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return idTokenClaims{}, err
	}
	return claims, nil
}

func ensureUser(ctx context.Context, db *Db, username string) (AuthUser, error) {
	row, err := db.Queries.GetUserByUsername(ctx, username)
	if err == nil {
		return AuthUser{ID: row.ID, Username: row.Username}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return AuthUser{}, err
	}

	created, err := db.Queries.CreateUser(ctx, db.CreateUserParams{
		Username: username,
		Password: "oauth",
	})
	if err != nil {
		return AuthUser{}, err
	}
	return AuthUser{ID: created.ID, Username: created.Username}, nil
}

func createSession(ctx context.Context, db *Db, userID int32) (string, time.Time, error) {
	expiresAt := time.Now().Add(sessionDuration)
	row, err := db.Queries.CreateSession(ctx, db.CreateSessionParams{
		UserID:  userID,
		Expires: pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
	if err != nil {
		return "", time.Time{}, err
	}
	return row.ID, expiresAt, nil
}

func getUserBySession(ctx context.Context, db *Db, sessionID string) (AuthUser, time.Time, error) {
	row, err := db.Queries.GetUserBySession(ctx, sessionID)
	if err != nil {
		return AuthUser{}, time.Time{}, err
	}
	return AuthUser{ID: row.ID, Username: row.Username}, row.Expires.Time, nil
}
