package response

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

type envelope struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *apiError   `json:"error,omitempty"`
	Meta    meta        `json:"meta"`
}

type apiError struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

type meta struct {
	RequestID string    `json:"request_id"`
	Timestamp time.Time `json:"timestamp"`
}

type problemDetails struct {
	Type      string `json:"type"`
	Title     string `json:"title"`
	Status    int    `json:"status"`
	Detail    string `json:"detail"`
	Instance  string `json:"instance"`
	Code      string `json:"code"`
	RequestID string `json:"request_id"`
}

func JSON(w http.ResponseWriter, r *http.Request, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(envelope{Success: true, Data: data, Meta: buildMeta(r)})
}

func Error(w http.ResponseWriter, r *http.Request, status int, code, message string, details interface{}) {
	if prefersProblemJSON(r) {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(problemDetails{
			Type:      problemType(code),
			Title:     problemTitle(code, status),
			Status:    status,
			Detail:    message,
			Instance:  r.URL.Path,
			Code:      code,
			RequestID: buildMeta(r).RequestID,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(envelope{
		Success: false,
		Error:   &apiError{Code: code, Message: message, Details: details},
		Meta:    buildMeta(r),
	})
}

func buildMeta(r *http.Request) meta {
	id := chimiddleware.GetReqID(r.Context())
	if id == "" {
		id = r.Header.Get("X-Request-Id")
	}
	if id == "" {
		id = "req-unknown"
	}
	return meta{RequestID: id, Timestamp: time.Now().UTC()}
}

func prefersProblemJSON(r *http.Request) bool {
	accept := strings.ToLower(r.Header.Get("Accept"))
	if accept == "" {
		return false
	}
	for _, part := range strings.Split(accept, ",") {
		item := strings.TrimSpace(part)
		if item == "" {
			continue
		}
		mediaType := item
		q := "1"
		if i := strings.Index(item, ";"); i >= 0 {
			mediaType = strings.TrimSpace(item[:i])
			params := strings.Split(item[i+1:], ";")
			for _, param := range params {
				p := strings.TrimSpace(param)
				if strings.HasPrefix(p, "q=") {
					q = strings.TrimSpace(strings.TrimPrefix(p, "q="))
				}
			}
		}
		if mediaType == "application/problem+json" && q != "0" && q != "0.0" && q != "0.00" && q != "0.000" {
			return true
		}
	}
	return false
}

func problemType(code string) string {
	normalized := strings.ToLower(strings.TrimSpace(code))
	normalized = strings.ReplaceAll(normalized, "_", "-")
	if normalized == "" {
		normalized = "unknown"
	}
	return "urn:problem:secure-observable:" + normalized
}

func problemTitle(code string, status int) string {
	switch strings.ToUpper(strings.TrimSpace(code)) {
	case "BAD_REQUEST":
		return "Bad Request"
	case "UNAUTHORIZED":
		return "Unauthorized"
	case "FORBIDDEN":
		return "Forbidden"
	case "CONFLICT":
		return "Conflict"
	case "NOT_FOUND":
		return "Not Found"
	case "INTERNAL":
		return "Internal Server Error"
	case "RATE_LIMITED":
		return "Too Many Requests"
	case "DEPENDENCY_UNREADY":
		return "Service Unavailable"
	case "INVALID_OR_EXPIRED_TOKEN":
		return "Invalid or Expired Token"
	case "EMAIL_UNVERIFIED":
		return "Email Unverified"
	default:
		if text := http.StatusText(status); text != "" {
			return text
		}
		return "Error"
	}
}
