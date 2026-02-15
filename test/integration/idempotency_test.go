package integration

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/sandeepkv93/everything-backend-starter-kit/internal/config"
)

func TestIdempotencyRegisterReplayAndConflict(t *testing.T) {
	baseURL, client, closeFn := newAuthTestServerWithOptions(t, authTestServerOptions{
		cfgOverride: func(cfg *config.Config) {
			cfg.IdempotencyEnabled = true
			cfg.IdempotencyRedisEnabled = false
		},
	})
	defer closeFn()

	payload := map[string]string{
		"email":    "idem-register@example.com",
		"name":     "Idempotent Register",
		"password": "Valid#Pass1234",
	}
	key := "idem-register-001"
	resp1, body1 := doRawText(t, client, http.MethodPost, baseURL+"/api/v1/auth/local/register", payload, map[string]string{
		"Idempotency-Key": key,
	}, nil)
	if resp1.StatusCode != http.StatusCreated {
		t.Fatalf("expected first register 201, got %d body=%q", resp1.StatusCode, body1)
	}

	resp2, body2 := doRawText(t, client, http.MethodPost, baseURL+"/api/v1/auth/local/register", payload, map[string]string{
		"Idempotency-Key": key,
	}, nil)
	if resp2.StatusCode != http.StatusCreated {
		t.Fatalf("expected replay register 201, got %d body=%q", resp2.StatusCode, body2)
	}
	if replayed := resp2.Header.Get("X-Idempotency-Replayed"); replayed != "true" {
		t.Fatalf("expected replay header, got %q", replayed)
	}
	if body1 != body2 {
		t.Fatalf("expected identical replay body\nfirst=%s\nsecond=%s", body1, body2)
	}

	conflictPayload := map[string]string{
		"email":    "idem-register-conflict@example.com",
		"name":     "Different Name",
		"password": "Valid#Pass1234",
	}
	resp3, env3 := doJSON(t, client, http.MethodPost, baseURL+"/api/v1/auth/local/register", conflictPayload, map[string]string{
		"Idempotency-Key": key,
	})
	if resp3.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 conflict, got %d", resp3.StatusCode)
	}
	if env3.Error == nil || env3.Error.Code != "CONFLICT" {
		t.Fatalf("expected CONFLICT envelope, got %#v", env3.Error)
	}
}

func TestIdempotencyMissingKeyRejectedOnScopedEndpoint(t *testing.T) {
	baseURL, client, closeFn := newAuthTestServerWithOptions(t, authTestServerOptions{
		cfgOverride: func(cfg *config.Config) {
			cfg.IdempotencyEnabled = true
			cfg.IdempotencyRedisEnabled = false
		},
	})
	defer closeFn()

	resp, body := doRawText(t, client, http.MethodPost, baseURL+"/api/v1/auth/local/register", map[string]string{
		"email":    "idem-missing-key@example.com",
		"name":     "Missing Key",
		"password": "Valid#Pass1234",
	}, nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing key, got %d", resp.StatusCode)
	}
	if !strings.Contains(body, "missing Idempotency-Key header") {
		t.Fatalf("expected missing key message, got %q", body)
	}
}

func TestIdempotencyAdminRoleCreateReplay(t *testing.T) {
	baseURL, client, closeFn := newAuthTestServerWithOptions(t, authTestServerOptions{
		cfgOverride: func(cfg *config.Config) {
			cfg.BootstrapAdminEmail = "idem-admin@example.com"
			cfg.IdempotencyEnabled = true
			cfg.IdempotencyRedisEnabled = false
		},
	})
	defer closeFn()

	registerResp, registerEnv := doJSON(t, client, http.MethodPost, baseURL+"/api/v1/auth/local/register", map[string]string{
		"email":    "idem-admin@example.com",
		"name":     "Idem Admin",
		"password": "Valid#Pass1234",
	}, map[string]string{
		"Idempotency-Key": "idem-admin-register-001",
	})
	if registerResp.StatusCode != http.StatusCreated || !registerEnv.Success {
		t.Fatalf("admin register failed: status=%d", registerResp.StatusCode)
	}

	loginResp, loginEnv := doJSON(t, client, http.MethodPost, baseURL+"/api/v1/auth/local/login", map[string]string{
		"email":    "idem-admin@example.com",
		"password": "Valid#Pass1234",
	}, nil)
	if loginResp.StatusCode != http.StatusOK || !loginEnv.Success {
		t.Fatalf("admin login failed: status=%d", loginResp.StatusCode)
	}

	roleBody := map[string]any{
		"name":        "idem-role",
		"description": "idempotency role",
		"permissions": []string{"users:read"},
	}
	key := "idem-admin-role-create-001"
	resp1, raw1 := doRawText(t, client, http.MethodPost, baseURL+"/api/v1/admin/roles", roleBody, map[string]string{
		"Idempotency-Key": key,
	}, nil)
	if resp1.StatusCode != http.StatusCreated {
		t.Fatalf("expected first create role 201, got %d body=%q", resp1.StatusCode, raw1)
	}

	resp2, raw2 := doRawText(t, client, http.MethodPost, baseURL+"/api/v1/admin/roles", roleBody, map[string]string{
		"Idempotency-Key": key,
	}, nil)
	if resp2.StatusCode != http.StatusCreated {
		t.Fatalf("expected replay create role 201, got %d body=%q", resp2.StatusCode, raw2)
	}
	if replayed := resp2.Header.Get("X-Idempotency-Replayed"); replayed != "true" {
		t.Fatalf("expected replay header, got %q", replayed)
	}

	var first map[string]any
	var second map[string]any
	if err := json.Unmarshal([]byte(raw1), &first); err != nil {
		t.Fatalf("decode first role response: %v", err)
	}
	if err := json.Unmarshal([]byte(raw2), &second); err != nil {
		t.Fatalf("decode second role response: %v", err)
	}
	if first["id"] != second["id"] {
		t.Fatalf("expected same role id on replay, got %v vs %v", first["id"], second["id"])
	}
}
