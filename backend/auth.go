package cashtrack

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	apiv1 "cashtrack/backend/gen/api/v1"
	dbgen "cashtrack/backend/gen/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type AuthHandler Handler

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

			sessionID, expiresAt, err := createSession(r.Context(), db, user.Id)
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

func ensureUser(ctx context.Context, db *Db, username string) (*apiv1.User, error) {
	row, err := db.Queries.GetUserByUsername(ctx, username)
	if err == nil {
		return &apiv1.User{Id: row.ID, Username: row.Username, Language: row.Language}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	created, err := db.Queries.CreateUser(ctx, dbgen.CreateUserParams{
		Username: username,
		Password: "oauth",
	})
	if err != nil {
		return nil, err
	}
	return &apiv1.User{Id: created.ID, Username: created.Username, Language: created.Language}, nil
}

func createSession(ctx context.Context, db *Db, userID int32) (string, time.Time, error) {
	expiresAt := time.Now().Add(sessionDuration)
	sessionID, err := db.Queries.CreateSession(ctx, dbgen.CreateSessionParams{
		UserID:  pgtype.Int4{Int32: userID, Valid: true},
		Expires: pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
	if err != nil {
		return "", time.Time{}, err
	}
	return sessionID, expiresAt, nil
}

func getUserBySession(ctx context.Context, db *Db, sessionID string) (*apiv1.User, time.Time, error) {
	sessionUUID, err := parseSessionID(sessionID)
	if err != nil {
		return nil, time.Time{}, err
	}
	row, err := db.Queries.GetUserBySession(ctx, sessionUUID)
	if err != nil {
		return nil, time.Time{}, err
	}
	return &apiv1.User{Id: row.ID, Username: row.Username, Language: row.Language}, row.Expires.Time, nil
}

func parseSessionID(sessionID string) (pgtype.UUID, error) {
	var sessionUUID pgtype.UUID
	if err := sessionUUID.Scan(sessionID); err != nil {
		return pgtype.UUID{}, err
	}
	return sessionUUID, nil
}
