package integration

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestProblemDetailsContentNegotiation_DefaultEnvelope(t *testing.T) {
	baseURL, client, closeFn := newAuthTestServer(t)
	defer closeFn()

	resp, env := doJSON(t, client, http.MethodGet, baseURL+"/api/v1/me", nil, nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
	if got := resp.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected application/json content type, got %q", got)
	}
	if env.Error == nil || env.Error.Code != "UNAUTHORIZED" {
		t.Fatalf("expected envelope UNAUTHORIZED, got %#v", env.Error)
	}
}

func TestProblemDetailsContentNegotiation_ProblemJSON(t *testing.T) {
	baseURL, client, closeFn := newAuthTestServer(t)
	defer closeFn()

	resp, body := doRawText(t, client, http.MethodGet, baseURL+"/api/v1/me", nil, map[string]string{
		"Accept": "application/problem+json",
	}, nil)
	assertProblemDetails(t, resp, body, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized", "/api/v1/me")
}

func TestProblemDetailsConsistencyFor400401403404(t *testing.T) {
	baseURL, client, closeFn := newAuthTestServer(t)
	defer closeFn()

	// 400
	resp, body := doRawText(t, client, http.MethodPost, baseURL+"/api/v1/auth/local/login", "oops", map[string]string{
		"Accept": "application/problem+json",
	}, nil)
	assertProblemDetails(t, resp, body, http.StatusBadRequest, "BAD_REQUEST", "Bad Request", "/api/v1/auth/local/login")

	// 401
	resp, body = doRawText(t, client, http.MethodGet, baseURL+"/api/v1/me", nil, map[string]string{
		"Accept": "application/problem+json",
	}, nil)
	assertProblemDetails(t, resp, body, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized", "/api/v1/me")

	registerAndLogin(t, client, baseURL, "problem-test@example.com", "Valid#Pass1234")

	// 403 CSRF
	resp, body = doRawText(t, client, http.MethodPost, baseURL+"/api/v1/auth/refresh", nil, map[string]string{
		"Accept": "application/problem+json",
	}, nil)
	assertProblemDetails(t, resp, body, http.StatusForbidden, "FORBIDDEN", "Forbidden", "/api/v1/auth/refresh")

	// 404
	csrf := cookieValue(t, client, baseURL, "csrf_token")
	resp, body = doRawText(t, client, http.MethodDelete, baseURL+"/api/v1/me/sessions/999999", nil, map[string]string{
		"Accept":       "application/problem+json",
		"X-CSRF-Token": csrf,
	}, nil)
	assertProblemDetails(t, resp, body, http.StatusNotFound, "NOT_FOUND", "Not Found", "/api/v1/me/sessions/999999")
}

func assertProblemDetails(t *testing.T, resp *http.Response, raw string, wantStatus int, wantCode, wantTitle, wantInstance string) {
	t.Helper()
	if resp.StatusCode != wantStatus {
		t.Fatalf("expected status %d, got %d body=%q", wantStatus, resp.StatusCode, raw)
	}
	if got := resp.Header.Get("Content-Type"); got != "application/problem+json" {
		t.Fatalf("expected application/problem+json, got %q body=%q", got, raw)
	}
	var p struct {
		Type      string `json:"type"`
		Title     string `json:"title"`
		Status    int    `json:"status"`
		Detail    string `json:"detail"`
		Instance  string `json:"instance"`
		Code      string `json:"code"`
		RequestID string `json:"request_id"`
	}
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		t.Fatalf("decode problem details: %v body=%q", err, raw)
	}
	if p.Status != wantStatus {
		t.Fatalf("unexpected status field: %d", p.Status)
	}
	if p.Code != wantCode {
		t.Fatalf("unexpected code field: %q", p.Code)
	}
	if p.Title != wantTitle {
		t.Fatalf("unexpected title field: %q", p.Title)
	}
	if p.Instance != wantInstance {
		t.Fatalf("unexpected instance field: %q", p.Instance)
	}
	if p.Type != "urn:problem:secure-observable:"+strings.ToLower(strings.ReplaceAll(wantCode, "_", "-")) {
		t.Fatalf("unexpected type field: %q", p.Type)
	}
	if p.RequestID == "" {
		t.Fatal("expected request_id in problem details")
	}
	if p.Detail == "" {
		t.Fatal("expected detail in problem details")
	}
}
