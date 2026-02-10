package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sandeepkv93/secure-observable-go-backend-starter-kit/internal/security"
)

type mockLimiter struct {
	allow bool
	retry time.Duration
	err   error
}

func (m mockLimiter) Allow(context.Context, string, int, time.Duration) (bool, time.Duration, error) {
	return m.allow, m.retry, m.err
}

type recordingLimiter struct {
	lastKey string
	allow   bool
}

func (r *recordingLimiter) Allow(_ context.Context, key string, _ int, _ time.Duration) (bool, time.Duration, error) {
	r.lastKey = key
	return r.allow, 0, nil
}

func TestDistributedRateLimiterFailOpenOnBackendError(t *testing.T) {
	rl := NewDistributedRateLimiter(
		mockLimiter{err: errors.New("redis down")},
		10,
		time.Minute,
		FailOpen,
		"api",
	)
	h := rl.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.RemoteAddr = "10.0.0.1:1111"
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected fail-open to allow request, got %d", rr.Code)
	}
}

func TestDistributedRateLimiterFailClosedOnBackendError(t *testing.T) {
	rl := NewDistributedRateLimiter(
		mockLimiter{err: errors.New("redis down")},
		10,
		time.Minute,
		FailClosed,
		"auth",
	)
	h := rl.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodPost, "/auth", nil)
	req.RemoteAddr = "10.0.0.1:1111"
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("expected fail-closed to reject request, got %d", rr.Code)
	}
}

func TestDistributedRateLimiterDeniedSetsRetryAfter(t *testing.T) {
	rl := NewDistributedRateLimiter(
		mockLimiter{allow: false, retry: 5 * time.Second},
		1,
		time.Minute,
		FailClosed,
		"api",
	)
	h := rl.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.RemoteAddr = "10.0.0.1:1111"
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rr.Code)
	}
	if got := rr.Header().Get("Retry-After"); got == "" {
		t.Fatal("expected Retry-After header")
	}
}

func TestSubjectOrIPKeyFuncUsesSubjectWhenAccessTokenValid(t *testing.T) {
	jwtMgr := security.NewJWTManager(
		"iss",
		"aud",
		"abcdefghijklmnopqrstuvwxyz123456",
		"abcdefghijklmnopqrstuvwxyz654321",
	)
	token, err := jwtMgr.SignAccessToken(42, nil, nil, 15*time.Minute)
	if err != nil {
		t.Fatalf("sign access token: %v", err)
	}

	limiter := &recordingLimiter{allow: true}
	rl := NewDistributedRateLimiterWithKey(
		limiter,
		10,
		time.Minute,
		FailClosed,
		"api",
		SubjectOrIPKeyFunc(jwtMgr),
	)

	h := rl.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.RemoteAddr = "10.0.0.1:1111"
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected request to pass, got %d", rr.Code)
	}
	if limiter.lastKey != "sub:42" {
		t.Fatalf("expected subject key, got %q", limiter.lastKey)
	}
}

func TestSubjectOrIPKeyFuncFallsBackToIPWhenTokenInvalid(t *testing.T) {
	jwtMgr := security.NewJWTManager(
		"iss",
		"aud",
		"abcdefghijklmnopqrstuvwxyz123456",
		"abcdefghijklmnopqrstuvwxyz654321",
	)
	limiter := &recordingLimiter{allow: true}
	rl := NewDistributedRateLimiterWithKey(
		limiter,
		10,
		time.Minute,
		FailClosed,
		"api",
		SubjectOrIPKeyFunc(jwtMgr),
	)

	h := rl.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.RemoteAddr = "10.0.0.1:1111"
	req.Header.Set("Authorization", "Bearer not-a-token")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected request to pass, got %d", rr.Code)
	}
	if limiter.lastKey != "10.0.0.1" {
		t.Fatalf("expected IP key fallback, got %q", limiter.lastKey)
	}
}
