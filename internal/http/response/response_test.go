package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestError_DefaultEnvelopeWhenProblemNotRequested(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("X-Request-Id", "req-test-1")
	rr := httptest.NewRecorder()

	Error(rr, req, http.StatusBadRequest, "BAD_REQUEST", "invalid payload", nil)

	if got := rr.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected application/json, got %q", got)
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if body["success"] != false {
		t.Fatalf("expected success=false, got %+v", body["success"])
	}
	if _, ok := body["meta"]; !ok {
		t.Fatalf("expected meta field in envelope: %+v", body)
	}
}

func TestError_ProblemDetailsWhenRequested(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/example/path", nil)
	req.Header.Set("Accept", "application/problem+json")
	req.Header.Set("X-Request-Id", "req-test-2")
	rr := httptest.NewRecorder()

	Error(rr, req, http.StatusUnauthorized, "UNAUTHORIZED", "missing access token", nil)

	if got := rr.Header().Get("Content-Type"); got != "application/problem+json" {
		t.Fatalf("expected application/problem+json, got %q", got)
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode problem details: %v", err)
	}
	if body["type"] != "urn:problem:secure-observable:unauthorized" {
		t.Fatalf("unexpected problem type: %+v", body["type"])
	}
	if body["title"] != "Unauthorized" {
		t.Fatalf("unexpected title: %+v", body["title"])
	}
	if body["status"] != float64(http.StatusUnauthorized) {
		t.Fatalf("unexpected status: %+v", body["status"])
	}
	if body["code"] != "UNAUTHORIZED" {
		t.Fatalf("unexpected code: %+v", body["code"])
	}
	if body["request_id"] != "req-test-2" {
		t.Fatalf("unexpected request_id: %+v", body["request_id"])
	}
	if body["instance"] != "/example/path" {
		t.Fatalf("unexpected instance: %+v", body["instance"])
	}
}

func TestError_ContentNegotiationVariants(t *testing.T) {
	tests := []struct {
		name   string
		accept string
		wantCT string
	}{
		{name: "jsonThenProblem", accept: "application/json, application/problem+json", wantCT: "application/problem+json"},
		{name: "problemWithQualityZero", accept: "application/problem+json;q=0", wantCT: "application/json"},
		{name: "missingAccept", accept: "", wantCT: "application/json"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/x", nil)
			if tc.accept != "" {
				req.Header.Set("Accept", tc.accept)
			}
			rr := httptest.NewRecorder()
			Error(rr, req, http.StatusBadRequest, "BAD_REQUEST", "bad input", nil)
			if got := rr.Header().Get("Content-Type"); got != tc.wantCT {
				t.Fatalf("expected %q, got %q", tc.wantCT, got)
			}
		})
	}
}

func TestError_StatusTypeCodeConsistencyForKeyStatuses(t *testing.T) {
	tests := []struct {
		status    int
		code      string
		wantType  string
		wantTitle string
	}{
		{status: http.StatusBadRequest, code: "BAD_REQUEST", wantType: "urn:problem:secure-observable:bad-request", wantTitle: "Bad Request"},
		{status: http.StatusUnauthorized, code: "UNAUTHORIZED", wantType: "urn:problem:secure-observable:unauthorized", wantTitle: "Unauthorized"},
		{status: http.StatusForbidden, code: "FORBIDDEN", wantType: "urn:problem:secure-observable:forbidden", wantTitle: "Forbidden"},
		{status: http.StatusNotFound, code: "NOT_FOUND", wantType: "urn:problem:secure-observable:not-found", wantTitle: "Not Found"},
		{status: http.StatusInternalServerError, code: "INTERNAL", wantType: "urn:problem:secure-observable:internal", wantTitle: "Internal Server Error"},
	}

	for _, tc := range tests {
		t.Run(tc.code, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/status-test", nil)
			req.Header.Set("Accept", "application/problem+json")
			req.Header.Set("X-Request-Id", "req-status")
			rr := httptest.NewRecorder()

			Error(rr, req, tc.status, tc.code, "detail", nil)

			var body map[string]any
			if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode problem details: %v", err)
			}
			if body["status"] != float64(tc.status) {
				t.Fatalf("unexpected status: %+v", body["status"])
			}
			if body["type"] != tc.wantType {
				t.Fatalf("unexpected type: %+v", body["type"])
			}
			if body["title"] != tc.wantTitle {
				t.Fatalf("unexpected title: %+v", body["title"])
			}
			if body["code"] != tc.code {
				t.Fatalf("unexpected code: %+v", body["code"])
			}
			if body["request_id"] != "req-status" {
				t.Fatalf("unexpected request_id: %+v", body["request_id"])
			}
		})
	}
}
