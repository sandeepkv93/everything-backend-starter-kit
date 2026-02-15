package observability

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/sandeepkv93/everything-backend-starter-kit/internal/config"
)

func TestRuntimeShutdownNilAndEmpty(t *testing.T) {
	var r *Runtime
	if err := r.Shutdown(context.Background()); err != nil {
		t.Fatalf("nil runtime shutdown: %v", err)
	}

	r = &Runtime{}
	if err := r.Shutdown(context.Background()); err != nil {
		t.Fatalf("empty runtime shutdown: %v", err)
	}
}

func TestInitRuntimeAllDisabled(t *testing.T) {
	cfg := &config.Config{
		OTELLogsEnabled:    false,
		OTELMetricsEnabled: false,
		OTELTracingEnabled: false,
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	r, err := InitRuntime(context.Background(), cfg, logger)
	if err != nil {
		t.Fatalf("init runtime disabled: %v", err)
	}
	if r == nil || r.MeterProvider == nil || r.TracerProvider == nil {
		t.Fatalf("expected runtime providers, got %+v", r)
	}
	if err := r.Shutdown(context.Background()); err != nil {
		t.Fatalf("runtime shutdown: %v", err)
	}
}

func TestInitRuntimeMetricsErrorBranch(t *testing.T) {
	cfg := &config.Config{
		OTELLogsEnabled:          false,
		OTELMetricsEnabled:       true,
		OTELTracingEnabled:       false,
		OTELExporterOTLPEndpoint: "%",
		OTELExporterOTLPInsecure: true,
		OTELServiceName:          "svc",
		OTELEnvironment:          "test",
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	if _, err := InitRuntime(context.Background(), cfg, logger); err == nil {
		t.Fatal("expected runtime init error from metrics exporter")
	}
}

func TestInitRuntimeTracingErrorBranch(t *testing.T) {
	cfg := &config.Config{
		OTELLogsEnabled:          false,
		OTELMetricsEnabled:       false,
		OTELTracingEnabled:       true,
		OTELExporterOTLPEndpoint: "%",
		OTELExporterOTLPInsecure: true,
		OTELServiceName:          "svc",
		OTELEnvironment:          "test",
		OTELTraceSamplingRatio:   1,
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	if _, err := InitRuntime(context.Background(), cfg, logger); err == nil {
		t.Fatal("expected runtime init error from tracing exporter")
	}
}
