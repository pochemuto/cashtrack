package cashtrack

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type AuthHandler Handler
type AuthMeHandler Handler

const sessionCookieName = "session_id"
const sessionDuration = 7 * 24 * time.Hour

type idTokenClaims struct {
	Sub   string `json:"sub"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

type authUser struct {
	ID       int32  `json:"id"`
	Username string `json:"username"`
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

func ensureUser(ctx context.Context, db *Db, username string) (authUser, error) {
	var user authUser
	err := db.conn.QueryRow(ctx, `SELECT id, username FROM users WHERE username = $1`, username).
		Scan(&user.ID, &user.Username)
	if err == nil {
		return user, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return authUser{}, err
	}

	err = db.conn.QueryRow(ctx, `INSERT INTO users (username, password) VALUES ($1, $2) RETURNING id, username`, username, "oauth").
		Scan(&user.ID, &user.Username)
	if err != nil {
		return authUser{}, err
	}
	return user, nil
}

func createSession(ctx context.Context, db *Db, userID int32) (string, time.Time, error) {
	expiresAt := time.Now().Add(sessionDuration)
	var sessionID string
	err := db.conn.QueryRow(ctx, `INSERT INTO sessions (user_id, expires) VALUES ($1, $2) RETURNING id`, userID, expiresAt).
		Scan(&sessionID)
	if err != nil {
		return "", time.Time{}, err
	}
	return sessionID, expiresAt, nil
}

func getUserBySession(ctx context.Context, db *Db, sessionID string) (authUser, time.Time, error) {
	var user authUser
	var expiresAt time.Time
	err := db.conn.QueryRow(ctx, `
		SELECT u.id, u.username, s.expires
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.id = $1
	`, sessionID).Scan(&user.ID, &user.Username, &expiresAt)
	if err != nil {
		return authUser{}, time.Time{}, err
	}
	return user, expiresAt, nil
}
