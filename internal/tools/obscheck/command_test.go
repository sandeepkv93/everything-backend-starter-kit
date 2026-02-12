package obscheck

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewRootCommandHasRun(t *testing.T) {
	cmd := NewRootCommand()
	if cmd.Use != "obscheck" {
		t.Fatalf("unexpected use: %s", cmd.Use)
	}
	if c, _, err := cmd.Find([]string{"run"}); err != nil || c == nil {
		t.Fatalf("expected run subcommand: err=%v", err)
	}
}

func TestGrafanaGETInvalidURLAndHTTPError(t *testing.T) {
	if _, err := grafanaGET(context.Background(), options{grafanaURL: "://bad"}, "/x"); err == nil {
		t.Fatal("expected parse url error")
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()
	if _, err := grafanaGET(context.Background(), options{grafanaURL: srv.URL}, "/x"); err == nil {
		t.Fatal("expected http status error")
	}
}

func TestFetchTraceIDFromExemplarNoRecentTrace(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer srv.Close()

	_, err := fetchTraceIDFromExemplar(context.Background(), options{grafanaURL: srv.URL, window: time.Minute}, time.Now().Add(-time.Minute))
	if err == nil {
		t.Fatal("expected no trace exemplar error")
	}
}

func TestFetchTraceIDFromExemplarFindsRecentTrace(t *testing.T) {
	now := time.Now().Unix()
	traceID := "0123456789abcdef0123456789abcdef"
	payload := fmt.Sprintf(`{"data":[{"exemplars":[{"timestamp":%d,"labels":{"trace_id":"%s"}}]}]}`, now, traceID)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(payload))
	}))
	defer srv.Close()

	got, err := fetchTraceIDFromExemplar(context.Background(), options{grafanaURL: srv.URL, window: time.Minute}, time.Now().Add(-time.Minute))
	if err != nil {
		t.Fatalf("fetch trace id: %v", err)
	}
	if got != traceID {
		t.Fatalf("unexpected trace id %q", got)
	}
}
