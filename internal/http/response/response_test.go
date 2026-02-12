package response

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"
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

func FuzzErrorContentNegotiationAndEnvelope(f *testing.F) {
	f.Add("application/problem+json", "/x", http.StatusBadRequest, "BAD_REQUEST", "invalid payload")
	f.Add("application/json, application/problem+json;q=0", "/api/v1/me", http.StatusUnauthorized, "UNAUTHORIZED", "missing token")
	f.Add("text/plain,application/problem+json;q=1.0", "/weird/\u2603", 499, "CUSTOM_CODE", "unicode \u2603 detail")
	f.Add(strings.Repeat("a", 2048), "/"+strings.Repeat("p", 128), http.StatusInternalServerError, "", strings.Repeat("m", 512))

	f.Fuzz(func(t *testing.T, accept, path string, status int, code, message string) {
		if len(accept) > 4096 {
			accept = accept[:4096]
		}
		if len(path) > 1024 {
			path = path[:1024]
		}
		if len(code) > 256 {
			code = code[:256]
		}
		if len(message) > 4096 {
			message = message[:4096]
		}
		codeIsValidUTF8 := utf8.ValidString(code)
		msgIsValidUTF8 := utf8.ValidString(message)
		path = strings.Map(func(r rune) rune {
			switch r {
			case ' ', '\t', '\n', '\r':
				return '_'
			default:
				if r < 0x20 || r == 0x7f {
					return '_'
				}
				return r
			}
		}, path)
		path = strings.ReplaceAll(path, "%", "_")
		path = strings.ReplaceAll(path, "?", "_")
		path = strings.ReplaceAll(path, "#", "_")
		if path == "" || !strings.HasPrefix(path, "/") {
			path = "/" + path
		}

		// Keep status in a writable HTTP range while still fuzzing edge values.
		status = ((status % 600) + 600) % 600
		if status < 100 {
			status += 100
		}

		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Accept", accept)
		req.Header.Set("X-Request-Id", "fuzz-req")
		rr := httptest.NewRecorder()

		Error(rr, req, status, code, message, map[string]any{"seed": "fuzz"})

		if rr.Code != status {
			t.Fatalf("status mismatch: got=%d want=%d", rr.Code, status)
		}

		ct := rr.Header().Get("Content-Type")
		if ct != "application/json" && ct != "application/problem+json" {
			t.Fatalf("unexpected content-type: %q", ct)
		}

		var body map[string]any
		if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
			t.Fatalf("invalid json body: %v, raw=%q", err, rr.Body.String())
		}

		if ct == "application/problem+json" {
			if got := int(body["status"].(float64)); got != status {
				t.Fatalf("problem status mismatch: got=%d want=%d", got, status)
			}
			gotCode, ok := body["code"].(string)
			if !ok {
				t.Fatalf("problem code must be string: %v", body["code"])
			}
			if codeIsValidUTF8 && gotCode != code {
				t.Fatalf("problem code mismatch: got=%v want=%v", gotCode, code)
			}
			if _, ok := body["request_id"]; !ok {
				t.Fatal("problem payload missing request_id")
			}
			if got, ok := body["instance"].(string); !ok || got != path {
				t.Fatalf("problem instance mismatch: got=%v want=%v", got, path)
			}
			return
		}

		if got, ok := body["success"].(bool); !ok || got {
			t.Fatalf("envelope success must be false, got=%v", body["success"])
		}
		errObj, ok := body["error"].(map[string]any)
		if !ok {
			t.Fatalf("missing error object in envelope: %v", body)
		}
		gotCode, ok := errObj["code"].(string)
		if !ok {
			t.Fatalf("envelope code must be string: %v", errObj["code"])
		}
		if codeIsValidUTF8 && gotCode != code {
			t.Fatalf("envelope code mismatch: got=%v want=%v", gotCode, code)
		}
		gotMessage, ok := errObj["message"].(string)
		if !ok {
			t.Fatalf("envelope message must be string: %v", errObj["message"])
		}
		if msgIsValidUTF8 && gotMessage != message {
			t.Fatalf("envelope message mismatch: got=%v want=%v", gotMessage, message)
		}
		if _, ok := body["meta"]; !ok {
			t.Fatalf("envelope missing meta: %s", fmt.Sprintf("%v", body))
		}
	})
}
