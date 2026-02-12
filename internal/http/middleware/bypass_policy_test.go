package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sandeepkv93/secure-observable-go-backend-starter-kit/internal/security"
)

func TestNewRequestBypassEvaluatorIgnoresInvalidCIDRsAndCanReturnNil(t *testing.T) {
	eval := NewRequestBypassEvaluator(RequestBypassConfig{
		EnableTrustedActorBypass: true,
		TrustedActorCIDRs:        []string{"not-a-cidr", "", "300.1.1.1/8"},
	}, nil)
	if eval != nil {
		t.Fatal("expected nil evaluator when trusted bypass has no valid cidrs/subjects and probes disabled")
	}
}

func TestRequestBypassEvaluatorMethodPathAndNilRequest(t *testing.T) {
	eval := NewRequestBypassEvaluator(RequestBypassConfig{EnableInternalProbeBypass: true}, nil)
	if eval == nil {
		t.Fatal("expected evaluator")
	}

	if bypass, reason := eval(nil); bypass || reason != "" {
		t.Fatalf("nil request should not bypass, got bypass=%v reason=%q", bypass, reason)
	}

	req := httptest.NewRequest(http.MethodPost, "/health/live", nil)
	if bypass, reason := eval(req); !bypass || reason != "internal_probe_path" {
		t.Fatalf("health/live should bypass regardless of method, got bypass=%v reason=%q", bypass, reason)
	}

	req = httptest.NewRequest(http.MethodGet, "/Health/Ready", nil)
	if bypass, reason := eval(req); !bypass || reason != "internal_probe_path" {
		t.Fatalf("path matching should be case-insensitive, got bypass=%v reason=%q", bypass, reason)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	if bypass, reason := eval(req); bypass || reason != "" {
		t.Fatalf("non-probe path should not bypass, got bypass=%v reason=%q", bypass, reason)
	}
}

func TestRequestBypassEvaluatorTrustedSubjectNormalizationAndFallback(t *testing.T) {
	jwtMgr := security.NewJWTManager(
		"iss",
		"aud",
		"abcdefghijklmnopqrstuvwxyz123456",
		"abcdefghijklmnopqrstuvwxyz654321",
	)
	tok, err := jwtMgr.SignAccessToken(7, nil, nil, time.Minute)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	eval := NewRequestBypassEvaluator(RequestBypassConfig{
		EnableTrustedActorBypass: true,
		TrustedActorSubjects:     []string{" 7 ", ""},
	}, jwtMgr)
	if eval == nil {
		t.Fatal("expected evaluator")
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	if bypass, reason := eval(req); !bypass || reason != "trusted_actor_subject" {
		t.Fatalf("expected trusted subject bypass, got bypass=%v reason=%q", bypass, reason)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
	req.Header.Set("Authorization", "Bearer bad.token")
	if bypass, reason := eval(req); bypass || reason != "" {
		t.Fatalf("invalid token should not bypass, got bypass=%v reason=%q", bypass, reason)
	}
}

func FuzzRequestBypassEvaluatorRobustness(f *testing.F) {
	f.Add(true, true, "/health/live", "GET", "203.0.113.10:1234", "", "", "7", true)
	f.Add(false, true, "/api/v1/admin/users", "POST", "198.51.100.2:8080", "198.51.100.0/24", "7", "7", true)
	f.Add(false, true, "/api/v1/admin/users", "POST", "bad-remote-addr", "bad-cidr", "", "", false)
	f.Add(true, false, "/api/v1/me", "PATCH", "", "", "", "", false)

	jwtMgr := security.NewJWTManager(
		"iss",
		"aud",
		"abcdefghijklmnopqrstuvwxyz123456",
		"abcdefghijklmnopqrstuvwxyz654321",
	)

	f.Fuzz(func(t *testing.T, enableProbe, enableTrusted bool, path, method, remoteAddr, cidr, trustedSubject, tokenSubject string, includeBearer bool) {
		if len(path) > 1024 {
			path = path[:1024]
		}
		if len(method) > 32 {
			method = method[:32]
		}
		if len(remoteAddr) > 128 {
			remoteAddr = remoteAddr[:128]
		}
		if len(cidr) > 128 {
			cidr = cidr[:128]
		}
		if len(trustedSubject) > 128 {
			trustedSubject = trustedSubject[:128]
		}
		if len(tokenSubject) > 128 {
			tokenSubject = tokenSubject[:128]
		}

		path = sanitizeFuzzPath(path)
		if method == "" {
			method = http.MethodGet
		}
		method = sanitizeFuzzMethod(method)

		cfg := RequestBypassConfig{
			EnableInternalProbeBypass: enableProbe,
			EnableTrustedActorBypass:  enableTrusted,
			TrustedActorCIDRs:         []string{cidr},
			TrustedActorSubjects:      []string{trustedSubject},
		}
		eval := NewRequestBypassEvaluator(cfg, jwtMgr)

		req := httptest.NewRequest(method, path, nil)
		req.RemoteAddr = strings.TrimSpace(remoteAddr)
		if includeBearer {
			sub := strings.TrimSpace(tokenSubject)
			if sub == "" {
				sub = "0"
			}
			tok, err := jwtMgr.SignAccessToken(7, nil, nil, time.Minute)
			if err != nil {
				t.Fatalf("sign token: %v", err)
			}
			// For subject fuzzing, replace with malformed token when non-numeric is requested.
			if sub != "7" {
				tok = "invalid.token.payload"
			}
			req.Header.Set("Authorization", "Bearer "+tok)
		}

		if eval == nil {
			return
		}
		b1, r1 := eval(req)
		b2, r2 := eval(req)
		if b1 != b2 || r1 != r2 {
			t.Fatalf("non-deterministic bypass result: first=(%v,%q) second=(%v,%q)", b1, r1, b2, r2)
		}
		switch r1 {
		case "", "internal_probe_path", "trusted_actor_cidr", "trusted_actor_subject":
		default:
			t.Fatalf("unexpected bypass reason %q", r1)
		}
	})
}
