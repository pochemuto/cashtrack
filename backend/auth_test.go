package cashtrack

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	apiv1 "cashtrack/backend/gen/api/v1"
	"cashtrack/backend/gen/api/v1/apiv1connect"
	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestAuthHandlerMissingCredential(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	handler := NewAuthHandler(db).Handler
	req := httptest.NewRequest(http.MethodGet, "/auth", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestAuthHandlerCreatesSessionAndRedirects(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	credential := fakeIDToken(t, idTokenClaims{
		Sub:   "sub-123",
		Email: "test@example.com",
		Name:  "Test User",
	})

	handler := NewAuthHandler(db).Handler
	req := httptest.NewRequest(http.MethodGet, "/auth?credential="+credential+"&redirect=/todo", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusFound {
		t.Fatalf("expected status 302, got %d", res.StatusCode)
	}
	if location := res.Header.Get("Location"); location != "/todo" {
		t.Fatalf("expected redirect to /todo, got %q", location)
	}

	var sessionCookie *http.Cookie
	for _, cookie := range res.Cookies() {
		if cookie.Name == sessionCookieName {
			sessionCookie = cookie
			break
		}
	}
	if sessionCookie == nil || sessionCookie.Value == "" {
		t.Fatalf("expected session cookie to be set")
	}

	var userID int32
	err := db.conn.QueryRow(context.Background(), `SELECT id FROM users WHERE username = $1`, "test@example.com").Scan(&userID)
	if err != nil {
		t.Fatalf("expected user to be created: %v", err)
	}

	var storedUserID int32
	err = db.conn.QueryRow(context.Background(), `SELECT user_id FROM sessions WHERE id = $1`, sessionCookie.Value).Scan(&storedUserID)
	if err != nil {
		t.Fatalf("expected session to be created: %v", err)
	}
	if storedUserID != userID {
		t.Fatalf("expected session user id %d, got %d", userID, storedUserID)
	}
}

func TestAuthMeHandler(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	userID := createUser(t, db, "me@example.com")
	sessionID := createSessionForUser(t, db, userID)

	handler := NewAuthServiceHandler(db)
	server := httptest.NewServer(handler.Handler)
	defer server.Close()

	client := connect.NewClient[apiv1.AuthMeRequest, apiv1.AuthMeResponse](
		server.Client(),
		server.URL+apiv1connect.AuthServiceMeProcedure,
	)
	req := connect.NewRequest(&apiv1.AuthMeRequest{})
	req.Header().Set("Cookie", fmt.Sprintf("%s=%s", sessionCookieName, sessionID))

	res, err := client.CallUnary(context.Background(), req)
	if err != nil {
		t.Fatalf("failed to call auth me: %v", err)
	}
	if res.Msg.User == nil {
		t.Fatalf("expected user in response")
	}
	if res.Msg.User.Id != userID || res.Msg.User.Username != "me@example.com" {
		t.Fatalf("unexpected user response: %+v", res.Msg.User)
	}
}

func TestAuthMeHandlerUnauthorized(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	handler := NewAuthServiceHandler(db)
	server := httptest.NewServer(handler.Handler)
	defer server.Close()

	client := connect.NewClient[apiv1.AuthMeRequest, apiv1.AuthMeResponse](
		server.Client(),
		server.URL+apiv1connect.AuthServiceMeProcedure,
	)
	req := connect.NewRequest(&apiv1.AuthMeRequest{})

	_, err := client.CallUnary(context.Background(), req)
	if err == nil {
		t.Fatalf("expected unauthenticated error")
	}
	var connectErr *connect.Error
	if !errors.As(err, &connectErr) {
		t.Fatalf("expected connect error, got %T", err)
	}
	if connectErr.Code() != connect.CodeUnauthenticated {
		t.Fatalf("expected unauthenticated, got %v", connectErr.Code())
	}
}

func TestAuthLogoutHandler(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	userID := createUser(t, db, "logout@example.com")
	sessionID := createSessionForUser(t, db, userID)

	handler := NewAuthServiceHandler(db)
	server := httptest.NewServer(handler.Handler)
	defer server.Close()

	client := connect.NewClient[apiv1.AuthLogoutRequest, apiv1.AuthLogoutResponse](
		server.Client(),
		server.URL+apiv1connect.AuthServiceLogoutProcedure,
	)
	req := connect.NewRequest(&apiv1.AuthLogoutRequest{})
	req.Header().Set("Cookie", fmt.Sprintf("%s=%s", sessionCookieName, sessionID))

	res, err := client.CallUnary(context.Background(), req)
	if err != nil {
		t.Fatalf("failed to call auth logout: %v", err)
	}

	var remaining int
	err = db.conn.QueryRow(context.Background(), `SELECT COUNT(*) FROM sessions WHERE id = $1`, sessionID).Scan(&remaining)
	if err != nil {
		t.Fatalf("failed to query sessions: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("expected session to be deleted")
	}

	cookies := (&http.Response{Header: res.Header()}).Cookies()
	var cleared *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == sessionCookieName {
			cleared = cookie
			break
		}
	}
	if cleared == nil {
		t.Fatalf("expected session cookie to be cleared")
	}
	if cleared.Value != "" {
		t.Fatalf("expected session cookie to be cleared, got %q", cleared.Value)
	}
	if !cleared.Expires.Before(time.Now()) {
		t.Fatalf("expected session cookie to expire in the past")
	}
}

func fakeIDToken(t *testing.T, claims idTokenClaims) string {
	t.Helper()
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none"}`))
	payloadBytes, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("failed to marshal claims: %v", err)
	}
	payload := base64.RawURLEncoding.EncodeToString(payloadBytes)
	return strings.Join([]string{header, payload, ""}, ".")
}

func openTestDB(t *testing.T) (*Db, func()) {
	t.Helper()

	connString := os.Getenv("TEST_DB_CONNECTION_STRING")
	if connString == "" {
		t.Skip("TEST_DB_CONNECTION_STRING is not set")
	}

	schemaName := fmt.Sprintf("auth_test_%d", time.Now().UnixNano())

	ctx := context.Background()
	setupConn, err := pgx.Connect(ctx, connString)
	if err != nil {
		t.Fatalf("failed to connect to db: %v", err)
	}
	defer setupConn.Close(ctx)

	if _, err := setupConn.Exec(ctx, "CREATE SCHEMA "+schemaName); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	if _, err := setupConn.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`); err != nil {
		t.Skipf("uuid-ossp extension unavailable: %v", err)
	}

	if _, err := setupConn.Exec(ctx, fmt.Sprintf("SET search_path TO %s", schemaName)); err != nil {
		t.Fatalf("failed to set search_path: %v", err)
	}

	if _, err := setupConn.Exec(ctx, `
		CREATE TABLE users (
			id integer GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
			username varchar(255) UNIQUE NOT NULL,
			password varchar(255) NOT NULL
		);
		CREATE TABLE sessions (
			id uuid NOT NULL DEFAULT uuid_generate_v4() PRIMARY KEY,
			user_id integer REFERENCES users(id) ON DELETE CASCADE,
			expires timestamp with time zone NOT NULL
		);
	`); err != nil {
		t.Fatalf("failed to create tables: %v", err)
	}

	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}
	config.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		_, err := conn.Exec(ctx, fmt.Sprintf("SET search_path TO %s", schemaName))
		return err
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}

	db, err := NewDB(pool)
	if err != nil {
		pool.Close()
		t.Fatalf("failed to create db: %v", err)
	}

	cleanup := func() {
		pool.Close()
		_, _ = setupConn.Exec(ctx, "DROP SCHEMA "+schemaName+" CASCADE")
	}

	return db, cleanup
}

func createUser(t *testing.T, db *Db, username string) int32 {
	t.Helper()
	var id int32
	err := db.conn.QueryRow(context.Background(), `INSERT INTO users (username, password) VALUES ($1, $2) RETURNING id`, username, "oauth").
		Scan(&id)
	if err != nil {
		t.Fatalf("failed to insert user: %v", err)
	}
	return id
}

func createSessionForUser(t *testing.T, db *Db, userID int32) string {
	t.Helper()
	var sessionID string
	expiresAt := time.Now().Add(24 * time.Hour)
	err := db.conn.QueryRow(context.Background(), `INSERT INTO sessions (user_id, expires) VALUES ($1, $2) RETURNING id`, userID, expiresAt).
		Scan(&sessionID)
	if err != nil {
		t.Fatalf("failed to insert session: %v", err)
	}
	return sessionID
}
