package observability

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/sandeepkv93/secure-observable-go-backend-starter-kit/internal/config"
)

func TestInitTracingDisabledBranch(t *testing.T) {
	cfg := &config.Config{OTELTracingEnabled: false}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	tp, err := InitTracing(context.Background(), cfg, logger)
	if err != nil {
		t.Fatalf("init tracing disabled: %v", err)
	}
	if tp == nil {
		t.Fatal("expected tracer provider")
	}
	_ = tp.Shutdown(context.Background())
}

func TestInitTracingExporterErrorBranch(t *testing.T) {
	cfg := &config.Config{
		OTELTracingEnabled:       true,
		OTELExporterOTLPEndpoint: "%",
		OTELExporterOTLPInsecure: true,
		OTELServiceName:          "svc",
		OTELEnvironment:          "test",
		OTELTraceSamplingRatio:   1.0,
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	if _, err := InitTracing(context.Background(), cfg, logger); err == nil {
		t.Fatal("expected tracing init error for invalid endpoint")
	}
}
