package database

import (
	"testing"

	"github.com/sandeepkv93/secure-observable-go-backend-starter-kit/internal/config"
)

func TestOpenInvalidDSN(t *testing.T) {
	cfg := &config.Config{DatabaseURL: "%"}
	if _, err := Open(cfg); err == nil {
		t.Fatal("expected postgres open error for invalid DSN")
	}
}
