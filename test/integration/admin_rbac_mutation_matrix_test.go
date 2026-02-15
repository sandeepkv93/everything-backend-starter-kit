package integration

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/sandeepkv93/everything-backend-starter-kit/internal/config"
	"github.com/sandeepkv93/everything-backend-starter-kit/internal/domain"
	"github.com/sandeepkv93/everything-backend-starter-kit/internal/service"
)

type permissionView struct {
	ID       uint   `json:"id"`
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

type permissionListPage struct {
	Items []permissionView `json:"items"`
}

type failingSetRolesUserService struct {
	delegate service.UserServiceInterface
}

func (s failingSetRolesUserService) GetByID(id uint) (*domain.User, []string, error) {
	return s.delegate.GetByID(id)
}

func (s failingSetRolesUserService) List() ([]domain.User, error) {
	return s.delegate.List()
}

func (s failingSetRolesUserService) SetRoles(userID uint, roleIDs []uint) error {
	return errors.New("forced SetRoles failure")
}

func TestAdminRoleUpdateMutationMatrix(t *testing.T) {
	t.Run("update role success and validation failures", func(t *testing.T) {
		baseURL, client, closeFn := newAuthTestServerWithOptions(t, authTestServerOptions{
			cfgOverride: func(cfg *config.Config) {
				cfg.BootstrapAdminEmail = "admin-role-matrix@example.com"
			},
		})
		defer closeFn()
		registerAndLogin(t, client, baseURL, "admin-role-matrix@example.com", "Valid#Pass1234")

		roleID := mustCreateRole(t, client, baseURL, "matrix-role-updatable", []string{"users:read"})
		resp, env := doJSON(t, client, http.MethodPatch, baseURL+"/api/v1/admin/roles/"+itoa(roleID), map[string]any{
			"name":        "matrix-role-updated",
			"description": "updated desc",
			"permissions": []string{"users:read", "roles:read"},
		}, nil)
		if resp.StatusCode != http.StatusOK || !env.Success {
			t.Fatalf("expected role update success, got status=%d success=%v", resp.StatusCode, env.Success)
		}
		var updated roleView
		if err := json.Unmarshal(env.Data, &updated); err != nil {
			t.Fatalf("decode updated role: %v", err)
		}
		if updated.Name != "matrix-role-updated" {
			t.Fatalf("expected updated role name, got %q", updated.Name)
		}

		resp, env = doJSON(t, client, http.MethodPatch, baseURL+"/api/v1/admin/roles/"+itoa(roleID), map[string]any{
			"permissions": []string{"bad-format"},
		}, nil)
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400 for invalid permission format, got %d", resp.StatusCode)
		}
		if env.Error == nil || env.Error.Code != "BAD_REQUEST" {
			t.Fatalf("expected BAD_REQUEST envelope, got %#v", env.Error)
		}

		resp, raw := doRawText(t, client, http.MethodPatch, baseURL+"/api/v1/admin/roles/"+itoa(roleID), "{", map[string]string{
			"Content-Type": "application/json",
		}, nil)
		var envFromRaw apiEnvelope
		_ = json.Unmarshal([]byte(raw), &envFromRaw)
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400 for invalid payload, got %d", resp.StatusCode)
		}
		if envFromRaw.Error == nil || envFromRaw.Error.Code != "BAD_REQUEST" {
			t.Fatalf("expected BAD_REQUEST envelope, got %#v", envFromRaw.Error)
		}
	})

	t.Run("update role unknown and protected rejections", func(t *testing.T) {
		baseURL, client, closeFn := newAuthTestServerWithOptions(t, authTestServerOptions{
			cfgOverride: func(cfg *config.Config) {
				cfg.BootstrapAdminEmail = "admin-role-protected@example.com"
				cfg.RBACProtectedRoles = []string{"matrix-protected-role"}
			},
		})
		defer closeFn()
		registerAndLogin(t, client, baseURL, "admin-role-protected@example.com", "Valid#Pass1234")

		resp, env := doJSON(t, client, http.MethodPatch, baseURL+"/api/v1/admin/roles/999999", map[string]any{
			"name": "does-not-exist",
		}, nil)
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("expected 404 for unknown role, got %d", resp.StatusCode)
		}
		if env.Error == nil || env.Error.Code != "NOT_FOUND" {
			t.Fatalf("expected NOT_FOUND envelope, got %#v", env.Error)
		}

		protectedRoleID := mustCreateRole(t, client, baseURL, "matrix-protected-role", []string{"users:read"})
		resp, env = doJSON(t, client, http.MethodPatch, baseURL+"/api/v1/admin/roles/"+itoa(protectedRoleID), map[string]any{
			"name": "matrix-protected-role-2",
		}, nil)
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("expected 403 for protected role update, got %d", resp.StatusCode)
		}
		if env.Error == nil || env.Error.Code != "FORBIDDEN" {
			t.Fatalf("expected FORBIDDEN envelope, got %#v", env.Error)
		}
	})

	t.Run("update role lockout prevention", func(t *testing.T) {
		baseURL, client, closeFn := newAuthTestServerWithOptions(t, authTestServerOptions{
			cfgOverride: func(cfg *config.Config) {
				cfg.BootstrapAdminEmail = "admin-role-lockout@example.com"
				cfg.RBACProtectedRoles = []string{}
			},
		})
		defer closeFn()
		registerAndLogin(t, client, baseURL, "admin-role-lockout@example.com", "Valid#Pass1234")

		meID := mustCurrentUserID(t, client, baseURL)
		lockoutRoleID := mustCreateRole(t, client, baseURL, "matrix-role-lockout", []string{"roles:write"})
		setUserRoles(t, client, baseURL, meID, []uint{lockoutRoleID})

		resp, env := doJSON(t, client, http.MethodPatch, baseURL+"/api/v1/admin/roles/"+itoa(lockoutRoleID), map[string]any{
			"permissions": []string{"users:read"},
		}, nil)
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("expected 403 lockout prevention on role update, got %d", resp.StatusCode)
		}
		if env.Error == nil || env.Error.Code != "FORBIDDEN" {
			t.Fatalf("expected FORBIDDEN envelope, got %#v", env.Error)
		}
	})
}

func TestAdminPermissionMutationMatrix(t *testing.T) {
	t.Run("create permission invalid format rejected", func(t *testing.T) {
		baseURL, client, closeFn := newAuthTestServerWithOptions(t, authTestServerOptions{
			cfgOverride: func(cfg *config.Config) {
				cfg.BootstrapAdminEmail = "admin-perm-create-invalid@example.com"
			},
		})
		defer closeFn()
		registerAndLogin(t, client, baseURL, "admin-perm-create-invalid@example.com", "Valid#Pass1234")

		resp, env := doJSON(t, client, http.MethodPost, baseURL+"/api/v1/admin/permissions", map[string]any{
			"resource": "bad resource",
			"action":   "read",
		}, nil)
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400 for invalid permission resource, got %d", resp.StatusCode)
		}
		if env.Error == nil || env.Error.Code != "BAD_REQUEST" {
			t.Fatalf("expected BAD_REQUEST envelope, got %#v", env.Error)
		}
	})

	t.Run("update permission success conflict and not found", func(t *testing.T) {
		baseURL, client, closeFn := newAuthTestServerWithOptions(t, authTestServerOptions{
			cfgOverride: func(cfg *config.Config) {
				cfg.BootstrapAdminEmail = "admin-perm-update@example.com"
			},
		})
		defer closeFn()
		registerAndLogin(t, client, baseURL, "admin-perm-update@example.com", "Valid#Pass1234")

		permA := mustCreatePermission(t, client, baseURL, "matrix_perm_update", "a")
		_ = mustCreatePermission(t, client, baseURL, "matrix_perm_update", "b")

		resp, env := doJSON(t, client, http.MethodPatch, baseURL+"/api/v1/admin/permissions/"+itoa(permA), map[string]any{
			"resource": "matrix_perm_update",
			"action":   "c",
		}, nil)
		if resp.StatusCode != http.StatusOK || !env.Success {
			t.Fatalf("expected permission update success, got status=%d success=%v", resp.StatusCode, env.Success)
		}

		resp, env = doJSON(t, client, http.MethodPatch, baseURL+"/api/v1/admin/permissions/"+itoa(permA), map[string]any{
			"resource": "matrix_perm_update",
			"action":   "b",
		}, nil)
		if resp.StatusCode != http.StatusConflict {
			t.Fatalf("expected 409 for permission update conflict, got %d", resp.StatusCode)
		}
		if env.Error == nil || env.Error.Code != "CONFLICT" {
			t.Fatalf("expected CONFLICT envelope, got %#v", env.Error)
		}

		resp, env = doJSON(t, client, http.MethodPatch, baseURL+"/api/v1/admin/permissions/999999", map[string]any{
			"resource": "matrix_perm_update",
			"action":   "z",
		}, nil)
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("expected 404 for permission update not found, got %d", resp.StatusCode)
		}
		if env.Error == nil || env.Error.Code != "NOT_FOUND" {
			t.Fatalf("expected NOT_FOUND envelope, got %#v", env.Error)
		}
	})

	t.Run("update and delete permission protected rejections", func(t *testing.T) {
		baseURL, client, closeFn := newAuthTestServerWithOptions(t, authTestServerOptions{
			cfgOverride: func(cfg *config.Config) {
				cfg.BootstrapAdminEmail = "admin-perm-protected@example.com"
				cfg.RBACProtectedPermissions = []string{"matrix_protected:write"}
			},
		})
		defer closeFn()
		registerAndLogin(t, client, baseURL, "admin-perm-protected@example.com", "Valid#Pass1234")

		protectedID := mustCreatePermission(t, client, baseURL, "matrix_protected", "write")
		resp, env := doJSON(t, client, http.MethodPatch, baseURL+"/api/v1/admin/permissions/"+itoa(protectedID), map[string]any{
			"resource": "matrix_protected",
			"action":   "write2",
		}, nil)
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("expected 403 for protected permission update, got %d", resp.StatusCode)
		}
		if env.Error == nil || env.Error.Code != "FORBIDDEN" {
			t.Fatalf("expected FORBIDDEN envelope, got %#v", env.Error)
		}

		resp, env = doJSON(t, client, http.MethodDelete, baseURL+"/api/v1/admin/permissions/"+itoa(protectedID), nil, nil)
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("expected 403 for protected permission delete, got %d", resp.StatusCode)
		}
		if env.Error == nil || env.Error.Code != "FORBIDDEN" {
			t.Fatalf("expected FORBIDDEN envelope, got %#v", env.Error)
		}
	})

	t.Run("update and delete permission lockout rejections", func(t *testing.T) {
		baseURL, client, closeFn := newAuthTestServerWithOptions(t, authTestServerOptions{
			cfgOverride: func(cfg *config.Config) {
				cfg.BootstrapAdminEmail = "admin-perm-lockout@example.com"
				cfg.RBACProtectedPermissions = []string{}
				cfg.RBACProtectedRoles = []string{}
			},
		})
		defer closeFn()
		registerAndLogin(t, client, baseURL, "admin-perm-lockout@example.com", "Valid#Pass1234")

		meID := mustCurrentUserID(t, client, baseURL)
		permWriteID := mustPermissionIDByToken(t, client, baseURL, "permissions", "write")
		lockoutRoleID := mustCreateRole(t, client, baseURL, "matrix-perm-lockout-role", []string{"permissions:write"})
		setUserRoles(t, client, baseURL, meID, []uint{lockoutRoleID})

		resp, env := doJSON(t, client, http.MethodPatch, baseURL+"/api/v1/admin/permissions/"+itoa(permWriteID), map[string]any{
			"resource": "permissions",
			"action":   "write2",
		}, nil)
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("expected 403 lockout prevention on permission update, got %d", resp.StatusCode)
		}
		if env.Error == nil || env.Error.Code != "FORBIDDEN" {
			t.Fatalf("expected FORBIDDEN envelope, got %#v", env.Error)
		}

		resp, env = doJSON(t, client, http.MethodDelete, baseURL+"/api/v1/admin/permissions/"+itoa(permWriteID), nil, nil)
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("expected 403 lockout prevention on permission delete, got %d", resp.StatusCode)
		}
		if env.Error == nil || env.Error.Code != "FORBIDDEN" {
			t.Fatalf("expected FORBIDDEN envelope, got %#v", env.Error)
		}
	})

	t.Run("delete permission success and not found", func(t *testing.T) {
		baseURL, client, closeFn := newAuthTestServerWithOptions(t, authTestServerOptions{
			cfgOverride: func(cfg *config.Config) {
				cfg.BootstrapAdminEmail = "admin-perm-delete@example.com"
			},
		})
		defer closeFn()
		registerAndLogin(t, client, baseURL, "admin-perm-delete@example.com", "Valid#Pass1234")

		permID := mustCreatePermission(t, client, baseURL, "matrix_delete", "x")
		resp, env := doJSON(t, client, http.MethodDelete, baseURL+"/api/v1/admin/permissions/"+itoa(permID), nil, nil)
		if resp.StatusCode != http.StatusOK || !env.Success {
			t.Fatalf("expected permission delete success, got status=%d success=%v", resp.StatusCode, env.Success)
		}

		resp, env = doJSON(t, client, http.MethodDelete, baseURL+"/api/v1/admin/permissions/999999", nil, nil)
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("expected 404 for permission delete not found, got %d", resp.StatusCode)
		}
		if env.Error == nil || env.Error.Code != "NOT_FOUND" {
			t.Fatalf("expected NOT_FOUND envelope, got %#v", env.Error)
		}
	})
}

func TestAdminSetUserRolesValidationAndServiceError(t *testing.T) {
	t.Run("invalid user id rejected", func(t *testing.T) {
		baseURL, client, closeFn := newAuthTestServerWithOptions(t, authTestServerOptions{
			cfgOverride: func(cfg *config.Config) {
				cfg.BootstrapAdminEmail = "admin-set-roles-invalid-id@example.com"
			},
		})
		defer closeFn()
		registerAndLogin(t, client, baseURL, "admin-set-roles-invalid-id@example.com", "Valid#Pass1234")

		resp, env := doJSON(t, client, http.MethodPatch, baseURL+"/api/v1/admin/users/not-a-number/roles", map[string]any{
			"role_ids": []uint{1},
		}, nil)
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400 invalid user id, got %d", resp.StatusCode)
		}
		if env.Error == nil || env.Error.Code != "BAD_REQUEST" {
			t.Fatalf("expected BAD_REQUEST envelope, got %#v", env.Error)
		}
	})

	t.Run("service error path returns internal", func(t *testing.T) {
		baseURL, client, closeFn := newAuthTestServerWithOptions(t, authTestServerOptions{
			cfgOverride: func(cfg *config.Config) {
				cfg.BootstrapAdminEmail = "admin-set-roles-error@example.com"
			},
			adminUserSvc: failingSetRolesUserService{
				delegate: service.NewUserService(nil, service.NewRBACService()),
			},
		})
		defer closeFn()
		registerAndLogin(t, client, baseURL, "admin-set-roles-error@example.com", "Valid#Pass1234")

		resp, env := doJSON(t, client, http.MethodPatch, baseURL+"/api/v1/admin/users/1/roles", map[string]any{
			"role_ids": []uint{1},
		}, nil)
		if resp.StatusCode != http.StatusInternalServerError {
			t.Fatalf("expected 500 for set roles service error, got %d", resp.StatusCode)
		}
		if env.Error == nil || env.Error.Code != "INTERNAL" {
			t.Fatalf("expected INTERNAL envelope, got %#v", env.Error)
		}
	})
}

func mustCurrentUserID(t *testing.T, client *http.Client, baseURL string) uint {
	t.Helper()
	resp, env := doJSON(t, client, http.MethodGet, baseURL+"/api/v1/me", nil, nil)
	if resp.StatusCode != http.StatusOK || !env.Success {
		t.Fatalf("load current user failed: status=%d success=%v", resp.StatusCode, env.Success)
	}
	var me struct {
		ID uint `json:"id"`
	}
	if err := json.Unmarshal(env.Data, &me); err != nil {
		t.Fatalf("decode me payload: %v", err)
	}
	if me.ID == 0 {
		t.Fatal("expected non-zero user id")
	}
	return me.ID
}

func mustCreateRole(t *testing.T, client *http.Client, baseURL, name string, permissions []string) uint {
	t.Helper()
	resp, env := doJSON(t, client, http.MethodPost, baseURL+"/api/v1/admin/roles", map[string]any{
		"name":        name,
		"description": "matrix role",
		"permissions": permissions,
	}, nil)
	if resp.StatusCode != http.StatusCreated || !env.Success {
		t.Fatalf("create role %q failed: status=%d success=%v", name, resp.StatusCode, env.Success)
	}
	var role roleView
	if err := json.Unmarshal(env.Data, &role); err != nil {
		t.Fatalf("decode created role: %v", err)
	}
	if role.ID == 0 {
		t.Fatalf("expected non-zero role id for %q", name)
	}
	return role.ID
}

func setUserRoles(t *testing.T, client *http.Client, baseURL string, userID uint, roleIDs []uint) {
	t.Helper()
	resp, env := doJSON(t, client, http.MethodPatch, baseURL+"/api/v1/admin/users/"+itoa(userID)+"/roles", map[string]any{
		"role_ids": roleIDs,
	}, nil)
	if resp.StatusCode != http.StatusOK || !env.Success {
		t.Fatalf("set roles failed: status=%d success=%v", resp.StatusCode, env.Success)
	}
}

func mustCreatePermission(t *testing.T, client *http.Client, baseURL, resource, action string) uint {
	t.Helper()
	resp, env := doJSON(t, client, http.MethodPost, baseURL+"/api/v1/admin/permissions", map[string]any{
		"resource": resource,
		"action":   action,
	}, nil)
	if resp.StatusCode != http.StatusCreated || !env.Success {
		t.Fatalf("create permission %s:%s failed: status=%d success=%v", resource, action, resp.StatusCode, env.Success)
	}
	var permission permissionView
	if err := json.Unmarshal(env.Data, &permission); err != nil {
		t.Fatalf("decode created permission: %v", err)
	}
	if permission.ID == 0 {
		t.Fatalf("expected non-zero permission id for %s:%s", resource, action)
	}
	return permission.ID
}

func mustPermissionIDByToken(t *testing.T, client *http.Client, baseURL, resource, action string) uint {
	t.Helper()
	query := "/api/v1/admin/permissions?resource=" + resource + "&action=" + action + "&page=1&page_size=50"
	resp, env := doJSON(t, client, http.MethodGet, baseURL+query, nil, nil)
	if resp.StatusCode != http.StatusOK || !env.Success {
		t.Fatalf("list permissions failed: status=%d success=%v", resp.StatusCode, env.Success)
	}
	var page permissionListPage
	if err := json.Unmarshal(env.Data, &page); err != nil {
		t.Fatalf("decode permissions page: %v", err)
	}
	for _, permission := range page.Items {
		if permission.Resource == resource && permission.Action == action {
			return permission.ID
		}
	}
	t.Fatalf("permission %s:%s not found in list", resource, action)
	return 0
}
