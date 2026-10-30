package security

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestInvalidExpiredAndForgedSessions(t *testing.T) {
	f := newFixture(t)
	if rec := req(t, f, http.MethodPost, "/api/auth/login", map[string]string{"username": f.name, "password": "wrong"}, nil, ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("invalid login status %d", rec.Code)
	}
	cookie, csrf := login(t, f)
	if rec := req(t, f, http.MethodGet, "/api/accounts", nil, &http.Cookie{Name: "session_id", Value: "forged"}, csrf); rec.Code != http.StatusUnauthorized {
		t.Fatalf("forged session status %d", rec.Code)
	}
	if rec := req(t, f, http.MethodGet, "/api/accounts?session_id="+cookie.Value, nil, nil, ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("query session status %d", rec.Code)
	}
	if _, err := f.pool.Exec(context.Background(), "UPDATE sessions SET expires_at=NOW()-INTERVAL '1 minute' WHERE user_id=$1", f.user); err != nil {
		t.Fatal(err)
	}
	if rec := req(t, f, http.MethodGet, "/api/accounts", nil, cookie, csrf); rec.Code != http.StatusUnauthorized {
		t.Fatalf("expired session status %d", rec.Code)
	}
}

func TestRateLimitCSRFInjectionAndLeakage(t *testing.T) {
	f := newFixture(t)
	for i := 0; i < 8; i++ {
		rec := req(t, f, http.MethodPost, "/api/auth/login", map[string]string{"username": f.name, "password": "wrong"}, nil, "")
		if i >= 5 && rec.Code != http.StatusTooManyRequests {
			t.Fatalf("attempt %d status %d", i, rec.Code)
		}
	}
	f.router = newRouter(t, f.pool)
	cookie, csrf := loginFresh(t, f)
	for _, username := range []string{"' OR '1'='1", "; DROP TABLE users --"} {
		rec := req(t, f, http.MethodPost, "/api/auth/login", map[string]string{"username": username, "password": "wrong"}, nil, "")
		if rec.Code >= 500 || containsInternals(rec.Body.String()) {
			t.Fatalf("injection login status %d: %s", rec.Code, rec.Body.String())
		}
	}
	if rec := req(t, f, http.MethodPost, "/api/beneficiaries", map[string]any{"destinationAccountId": f.destination, "nickname": "x"}, cookie, ""); rec.Code != http.StatusForbidden {
		t.Fatalf("missing csrf status %d", rec.Code)
	}
	for _, target := range []string{
		"/api/accounts/" + f.account.String() + "/transactions?sort=%27%20OR%20%271%27=%271",
		"/api/accounts/" + f.account.String() + "/transactions?type=1%3BDROP%20TABLE%20users",
	} {
		rec := req(t, f, http.MethodGet, target, nil, cookie, csrf)
		if rec.Code >= 500 || containsInternals(rec.Body.String()) {
			t.Fatalf("unsafe response %s: %d %s", target, rec.Code, rec.Body.String())
		}
	}
	rec := req(t, f, http.MethodPost, "/api/beneficiaries", map[string]any{"destinationAccountId": f.destination, "nickname": "<script>alert(1)</script>"}, cookie, csrf)
	if rec.Code != http.StatusCreated {
		t.Fatalf("xss fixture status %d: %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Data["nickname"] != "<script>alert(1)</script>" {
		t.Fatal("XSS payload was not returned as inert JSON text")
	}
	transfer := transferReq(t, f, cookie, csrf, "security-xss-transfer-01", "<script>alert(2)</script>")
	if transfer.Code != http.StatusCreated {
		t.Fatalf("transfer XSS status %d: %s", transfer.Code, transfer.Body.String())
	}
	var transferBody struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(transfer.Body.Bytes(), &transferBody); err != nil {
		t.Fatal(err)
	}
	if transferBody.Data["purpose"] != "<script>alert(2)</script>" {
		t.Fatal("transfer purpose was not returned as inert JSON text")
	}
	if containsInternals(rec.Body.String()) || containsSensitive(rec.Body.String(), cookie.Value) {
		t.Fatal("sensitive data leaked in response")
	}
}

func loginFresh(t *testing.T, f *fixture) (*http.Cookie, string) {
	t.Helper()
	if _, err := f.pool.Exec(context.Background(), "DELETE FROM sessions WHERE user_id=$1", f.user); err != nil {
		t.Fatal(err)
	}
	return login(t, f)
}

func containsLiteral(value, literal string) bool { return strings.Contains(value, literal) }
func containsInternals(value string) bool {
	return strings.Contains(value, "stack trace") || strings.Contains(value, "SQLSTATE") || strings.Contains(value, "pq:")
}
func containsSensitive(value, session string) bool {
	return strings.Contains(value, session) || strings.Contains(value, "$argon2")
}
