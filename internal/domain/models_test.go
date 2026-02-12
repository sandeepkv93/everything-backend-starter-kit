package domain

import (
	"reflect"
	"strings"
	"testing"
)

func TestUserModelTagsAndDefaults(t *testing.T) {
	typ := reflect.TypeOf(User{})

	email, ok := typ.FieldByName("Email")
	if !ok {
		t.Fatal("missing User.Email field")
	}
	if got := email.Tag.Get("json"); got != "email" {
		t.Fatalf("User.Email json tag mismatch: %q", got)
	}
	if !strings.Contains(email.Tag.Get("gorm"), "uniqueIndex") {
		t.Fatalf("User.Email gorm tag missing uniqueIndex: %q", email.Tag.Get("gorm"))
	}

	status, ok := typ.FieldByName("Status")
	if !ok {
		t.Fatal("missing User.Status field")
	}
	statusTag := status.Tag.Get("gorm")
	if !strings.Contains(statusTag, "default:active") {
		t.Fatalf("User.Status gorm tag missing default:active: %q", statusTag)
	}
	if got := status.Tag.Get("json"); got != "status" {
		t.Fatalf("User.Status json tag mismatch: %q", got)
	}

	roles, ok := typ.FieldByName("Roles")
	if !ok {
		t.Fatal("missing User.Roles field")
	}
	if got := roles.Tag.Get("json"); got != "roles,omitempty" {
		t.Fatalf("User.Roles json tag mismatch: %q", got)
	}
	if !strings.Contains(roles.Tag.Get("gorm"), "many2many:user_roles") {
		t.Fatalf("User.Roles gorm tag missing many2many:user_roles: %q", roles.Tag.Get("gorm"))
	}
}

func TestRoleAndPermissionModelContracts(t *testing.T) {
	roleType := reflect.TypeOf(Role{})
	name, ok := roleType.FieldByName("Name")
	if !ok {
		t.Fatal("missing Role.Name field")
	}
	if !strings.Contains(name.Tag.Get("gorm"), "uniqueIndex") {
		t.Fatalf("Role.Name should be unique indexed: %q", name.Tag.Get("gorm"))
	}

	perms, ok := roleType.FieldByName("Permissions")
	if !ok {
		t.Fatal("missing Role.Permissions field")
	}
	if !strings.Contains(perms.Tag.Get("gorm"), "many2many:role_permissions") {
		t.Fatalf("Role.Permissions gorm tag mismatch: %q", perms.Tag.Get("gorm"))
	}

	permType := reflect.TypeOf(Permission{})
	res, ok := permType.FieldByName("Resource")
	if !ok {
		t.Fatal("missing Permission.Resource field")
	}
	if !strings.Contains(res.Tag.Get("gorm"), "idx_perm_unique,unique") {
		t.Fatalf("Permission.Resource gorm tag missing unique pair index: %q", res.Tag.Get("gorm"))
	}
	action, ok := permType.FieldByName("Action")
	if !ok {
		t.Fatal("missing Permission.Action field")
	}
	if !strings.Contains(action.Tag.Get("gorm"), "idx_perm_unique,unique") {
		t.Fatalf("Permission.Action gorm tag missing unique pair index: %q", action.Tag.Get("gorm"))
	}
}

func TestSensitiveFieldsAreHiddenFromJSON(t *testing.T) {
	cases := []struct {
		typeName string
		typ      reflect.Type
		field    string
	}{
		{typeName: "LocalCredential", typ: reflect.TypeOf(LocalCredential{}), field: "PasswordHash"},
		{typeName: "Session", typ: reflect.TypeOf(Session{}), field: "RefreshTokenHash"},
		{typeName: "Session", typ: reflect.TypeOf(Session{}), field: "TokenID"},
		{typeName: "VerificationToken", typ: reflect.TypeOf(VerificationToken{}), field: "TokenHash"},
		{typeName: "IdempotencyRecord", typ: reflect.TypeOf(IdempotencyRecord{}), field: "Scope"},
	}

	for _, tc := range cases {
		f, ok := tc.typ.FieldByName(tc.field)
		if !ok {
			t.Fatalf("%s.%s missing", tc.typeName, tc.field)
		}
		if got := f.Tag.Get("json"); got != "-" {
			t.Fatalf("expected %s.%s json tag '-' for sensitive field, got %q", tc.typeName, tc.field, got)
		}
	}
}

func TestSessionAndVerificationTokenIndexContracts(t *testing.T) {
	sessionType := reflect.TypeOf(Session{})
	expires, ok := sessionType.FieldByName("ExpiresAt")
	if !ok {
		t.Fatal("missing Session.ExpiresAt")
	}
	if !strings.Contains(expires.Tag.Get("gorm"), "index") {
		t.Fatalf("Session.ExpiresAt should be indexed: %q", expires.Tag.Get("gorm"))
	}
	revoked, ok := sessionType.FieldByName("RevokedAt")
	if !ok {
		t.Fatal("missing Session.RevokedAt")
	}
	if !strings.Contains(revoked.Tag.Get("gorm"), "index") {
		t.Fatalf("Session.RevokedAt should be indexed: %q", revoked.Tag.Get("gorm"))
	}

	vtType := reflect.TypeOf(VerificationToken{})
	expiresTok, ok := vtType.FieldByName("ExpiresAt")
	if !ok {
		t.Fatal("missing VerificationToken.ExpiresAt")
	}
	if !strings.Contains(expiresTok.Tag.Get("gorm"), "index") {
		t.Fatalf("VerificationToken.ExpiresAt should be indexed: %q", expiresTok.Tag.Get("gorm"))
	}
}

func TestAssociationJoinModelsHaveCompositePrimaryKeys(t *testing.T) {
	checkCompositePK := func(name string, typ reflect.Type, fields ...string) {
		t.Helper()
		for _, field := range fields {
			f, ok := typ.FieldByName(field)
			if !ok {
				t.Fatalf("missing %s.%s", name, field)
			}
			if !strings.Contains(f.Tag.Get("gorm"), "primaryKey") {
				t.Fatalf("expected %s.%s to be primaryKey, got %q", name, field, f.Tag.Get("gorm"))
			}
		}
	}

	checkCompositePK("UserRole", reflect.TypeOf(UserRole{}), "UserID", "RoleID")
	checkCompositePK("RolePermission", reflect.TypeOf(RolePermission{}), "RoleID", "PermissionID")
}
