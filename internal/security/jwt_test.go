package security

import (
	"strings"
	"testing"
	"time"
)

func TestJWTAccessAndRefreshParsing(t *testing.T) {
	mgr := NewJWTManager("iss", "aud", "abcdefghijklmnopqrstuvwxyz123456", "abcdefghijklmnopqrstuvwxyz654321")
	access, err := mgr.SignAccessToken(42, []string{"admin"}, []string{"users:read"}, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	refresh, err := mgr.SignRefreshToken(42, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	ac, err := mgr.ParseAccessToken(access)
	if err != nil {
		t.Fatal(err)
	}
	if ac.Subject != "42" || ac.TokenType != "access" {
		t.Fatalf("unexpected access claims: %+v", ac)
	}
	if _, err := mgr.ParseAccessToken(refresh); err == nil {
		t.Fatal("expected refresh token to fail access parse")
	}
}

func FuzzParseAccessTokenRobustness(f *testing.F) {
	mgr := NewJWTManager("iss", "aud", "abcdefghijklmnopqrstuvwxyz123456", "abcdefghijklmnopqrstuvwxyz654321")
	validAccess, _ := mgr.SignAccessToken(42, []string{"admin"}, []string{"users:read"}, time.Minute)
	validRefresh, _ := mgr.SignRefreshToken(42, time.Minute)

	f.Add(validAccess)
	f.Add(validRefresh)
	f.Add("")
	f.Add("not-a-jwt")
	f.Add("🔥.🔥.🔥")
	f.Add(strings.Repeat("a", 8192))

	f.Fuzz(func(t *testing.T, raw string) {
		if len(raw) > 16384 {
			raw = raw[:16384]
		}
		claims, err := mgr.ParseAccessToken(raw)
		if err == nil {
			if claims == nil {
				t.Fatal("expected non-nil claims on successful parse")
			}
			if claims.TokenType != "access" {
				t.Fatalf("unexpected token type: %q", claims.TokenType)
			}
			if claims.Subject == "" {
				t.Fatal("expected non-empty subject on successful parse")
			}
			return
		}
		if claims != nil && claims.TokenType == "access" && claims.Subject != "" {
			// partial claims may exist on parse error; ensure they are not trusted as success.
			_ = claims
		}
	})
}

func FuzzParseRefreshTokenRobustness(f *testing.F) {
	mgr := NewJWTManager("iss", "aud", "abcdefghijklmnopqrstuvwxyz123456", "abcdefghijklmnopqrstuvwxyz654321")
	validAccess, _ := mgr.SignAccessToken(42, []string{"admin"}, []string{"users:read"}, time.Minute)
	validRefresh, _ := mgr.SignRefreshToken(42, time.Minute)

	f.Add(validRefresh)
	f.Add(validAccess)
	f.Add("")
	f.Add("header.payload.signature")
	f.Add("こんにちは.世界")
	f.Add(strings.Repeat("b", 8192))

	f.Fuzz(func(t *testing.T, raw string) {
		if len(raw) > 16384 {
			raw = raw[:16384]
		}
		claims, err := mgr.ParseRefreshToken(raw)
		if err == nil {
			if claims == nil {
				t.Fatal("expected non-nil claims on successful parse")
			}
			if claims.TokenType != "refresh" {
				t.Fatalf("unexpected token type: %q", claims.TokenType)
			}
			if claims.Subject == "" {
				t.Fatal("expected non-empty subject on successful parse")
			}
			return
		}
		if claims != nil && claims.TokenType == "refresh" && claims.Subject != "" {
			_ = claims
		}
	})
}
