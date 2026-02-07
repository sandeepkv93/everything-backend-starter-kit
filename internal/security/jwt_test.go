package security

import (
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
