package di

import (
	"testing"

	"go-oauth-rbac-service/internal/config"
	"go-oauth-rbac-service/internal/http/router"
)

func TestProvideHTTPServer(t *testing.T) {
	cfg := &config.Config{HTTPPort: "9999"}
	srv := provideHTTPServer(cfg, nil)
	if srv.Addr != ":9999" {
		t.Fatalf("unexpected addr: %s", srv.Addr)
	}
	if srv.ReadTimeout.Seconds() != 10 {
		t.Fatalf("unexpected read timeout: %v", srv.ReadTimeout)
	}
}

func TestProvideRouterDependencies(t *testing.T) {
	cfg := &config.Config{CORSAllowedOrigins: []string{"http://localhost:3000"}, AuthRateLimitPerMin: 10, APIRateLimitPerMin: 100}
	dep := provideRouterDependencies(nil, nil, nil, nil, nil, cfg)
	if dep.AuthRateLimitRPM != 10 || dep.APIRateLimitRPM != 100 {
		t.Fatalf("unexpected rate limits: %+v", dep)
	}
	if len(dep.CORSOrigins) != 1 || dep.CORSOrigins[0] != "http://localhost:3000" {
		t.Fatalf("unexpected cors origins: %+v", dep.CORSOrigins)
	}
	_ = router.Dependencies(dep)
}
