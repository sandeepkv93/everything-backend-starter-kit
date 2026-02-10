package integration

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/sandeepkv93/secure-observable-go-backend-starter-kit/internal/http/middleware"
	"github.com/sandeepkv93/secure-observable-go-backend-starter-kit/internal/security"
)

func TestRateLimiterBlocksAfterLimit(t *testing.T) {
	rl := middleware.NewRateLimiter(2, time.Minute)
	r := chi.NewRouter()
	r.With(rl.Middleware()).Get("/x", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 on request %d, got %d", i+1, w.Code)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 got %d", w.Code)
	}
}

func TestRateLimiterSubjectKeyingAcrossIPs(t *testing.T) {
	jwtMgr := security.NewJWTManager(
		"iss",
		"aud",
		"abcdefghijklmnopqrstuvwxyz123456",
		"abcdefghijklmnopqrstuvwxyz654321",
	)
	subjectLimiter := middleware.NewRateLimiterWithKey(2, time.Minute, middleware.SubjectOrIPKeyFunc(jwtMgr))
	tokenUser1, err := jwtMgr.SignAccessToken(101, nil, nil, 15*time.Minute)
	if err != nil {
		t.Fatalf("sign token user1: %v", err)
	}
	tokenUser2, err := jwtMgr.SignAccessToken(202, nil, nil, 15*time.Minute)
	if err != nil {
		t.Fatalf("sign token user2: %v", err)
	}

	r := chi.NewRouter()
	r.With(subjectLimiter.Middleware()).Get("/x", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	req1 := httptest.NewRequest(http.MethodGet, "/x", nil)
	req1.RemoteAddr = "10.0.0.1:1234"
	req1.Header.Set("Authorization", "Bearer "+tokenUser1)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf("expected first user1 request 200, got %d", w1.Code)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/x", nil)
	req2.RemoteAddr = "10.0.0.2:1234"
	req2.Header.Set("Authorization", "Bearer "+tokenUser1)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("expected second user1 request from different IP 200, got %d", w2.Code)
	}

	req3 := httptest.NewRequest(http.MethodGet, "/x", nil)
	req3.RemoteAddr = "10.0.0.3:1234"
	req3.Header.Set("Authorization", "Bearer "+tokenUser1)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusTooManyRequests {
		t.Fatalf("expected user1 third request to be limited, got %d", w3.Code)
	}

	req4 := httptest.NewRequest(http.MethodGet, "/x", nil)
	req4.RemoteAddr = "10.0.0.1:1234"
	req4.Header.Set("Authorization", "Bearer "+tokenUser2)
	w4 := httptest.NewRecorder()
	r.ServeHTTP(w4, req4)
	if w4.Code != http.StatusOK {
		t.Fatalf("expected different user on same IP to have separate quota, got %d", w4.Code)
	}
}
