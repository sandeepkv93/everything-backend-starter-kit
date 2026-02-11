package observability

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"go.opentelemetry.io/otel/trace"
)

type testHandler struct {
	enabled     bool
	handleErr   error
	lastRecord  slog.Record
	handled     int
	groupPrefix string
	attrs       []slog.Attr
}

func (h *testHandler) Enabled(context.Context, slog.Level) bool { return h.enabled }

func (h *testHandler) Handle(_ context.Context, r slog.Record) error {
	h.handled++
	h.lastRecord = r
	return h.handleErr
}

func (h *testHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := *h
	next.attrs = append(append([]slog.Attr(nil), h.attrs...), attrs...)
	return &next
}

func (h *testHandler) WithGroup(name string) slog.Handler {
	next := *h
	next.groupPrefix = name
	return &next
}

func TestParseLogLevel(t *testing.T) {
	cases := map[string]slog.Level{
		"debug": slog.LevelDebug,
		"warn":  slog.LevelWarn,
		"error": slog.LevelError,
		"info":  slog.LevelInfo,
		"":      slog.LevelInfo,
	}
	for in, want := range cases {
		if got := parseLogLevel(in); got != want {
			t.Fatalf("parseLogLevel(%q)=%v want=%v", in, got, want)
		}
	}
}

func TestMultiHandlerEnabledAndHandle(t *testing.T) {
	h1 := &testHandler{enabled: false}
	h2 := &testHandler{enabled: true}
	mh := &multiHandler{handlers: []slog.Handler{h1, h2}}

	if !mh.Enabled(context.Background(), slog.LevelInfo) {
		t.Fatal("expected enabled when one child is enabled")
	}

	rec := slog.NewRecord(nowForTest(), slog.LevelInfo, "hello", 0)
	if err := mh.Handle(context.Background(), rec); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if h1.handled != 1 || h2.handled != 1 {
		t.Fatalf("expected both handlers invoked, got h1=%d h2=%d", h1.handled, h2.handled)
	}
}

func TestTraceContextHandlerAddsTraceFields(t *testing.T) {
	inner := &testHandler{enabled: true}
	h := &traceContextHandler{next: inner}

	rec := slog.NewRecord(nowForTest(), slog.LevelInfo, "msg", 0)
	if err := h.Handle(context.Background(), rec); err != nil {
		t.Fatalf("handle no span: %v", err)
	}
	attrs := attrsToMap(inner.lastRecord)
	if attrs["trace_id"] != "" || attrs["span_id"] != "" {
		t.Fatalf("expected empty trace attrs, got trace_id=%q span_id=%q", attrs["trace_id"], attrs["span_id"])
	}

	traceID, _ := trace.TraceIDFromHex("0102030405060708090a0b0c0d0e0f10")
	spanID, _ := trace.SpanIDFromHex("0102030405060708")
	sc := trace.NewSpanContext(trace.SpanContextConfig{TraceID: traceID, SpanID: spanID, TraceFlags: trace.FlagsSampled})
	ctx := trace.ContextWithSpanContext(context.Background(), sc)
	rec2 := slog.NewRecord(nowForTest(), slog.LevelInfo, "msg2", 0)
	if err := h.Handle(ctx, rec2); err != nil {
		t.Fatalf("handle with span: %v", err)
	}
	attrs = attrsToMap(inner.lastRecord)
	if attrs["trace_id"] == "" || attrs["span_id"] == "" {
		t.Fatalf("expected trace attrs to be populated, got trace_id=%q span_id=%q", attrs["trace_id"], attrs["span_id"])
	}
}

func attrsToMap(rec slog.Record) map[string]string {
	out := map[string]string{}
	rec.Attrs(func(a slog.Attr) bool {
		out[a.Key] = a.Value.String()
		return true
	})
	return out
}

func nowForTest() time.Time {
	return time.Unix(1700000000, 0).UTC()
}
