package repository

import (
	"errors"
	"testing"

	"github.com/sandeepkv93/secure-observable-go-backend-starter-kit/internal/domain"
)

func TestRoleRepositoryCreateUpdateDeleteAndConflict(t *testing.T) {
	db := newRepositoryDBForTest(t)
	roleRepo := NewRoleRepository(db)
	permRepo := NewPermissionRepository(db)

	permA := &domain.Permission{Resource: "users", Action: "read"}
	permB := &domain.Permission{Resource: "users", Action: "write"}
	for _, p := range []*domain.Permission{permA, permB} {
		if err := permRepo.Create(p); err != nil {
			t.Fatalf("create permission: %v", err)
		}
	}

	role := &domain.Role{Name: "manager", Description: "can manage users"}
	if err := roleRepo.Create(role, []uint{permA.ID}); err != nil {
		t.Fatalf("create role: %v", err)
	}
	created, err := roleRepo.FindByID(role.ID)
	if err != nil {
		t.Fatalf("find created role: %v", err)
	}
	if len(created.Permissions) != 1 || created.Permissions[0].ID != permA.ID {
		t.Fatalf("expected one permission bound on create, got %+v", created.Permissions)
	}

	if err := roleRepo.Update(&domain.Role{ID: role.ID, Name: "manager-updated", Description: "updated"}, []uint{permB.ID}); err != nil {
		t.Fatalf("update role: %v", err)
	}
	updated, err := roleRepo.FindByID(role.ID)
	if err != nil {
		t.Fatalf("find updated role: %v", err)
	}
	if updated.Name != "manager-updated" || len(updated.Permissions) != 1 || updated.Permissions[0].ID != permB.ID {
		t.Fatalf("unexpected updated role: %+v", updated)
	}

	if err := roleRepo.Update(&domain.Role{ID: 999999, Name: "missing"}, nil); !errors.Is(err, ErrRoleNotFound) {
		t.Fatalf("expected ErrRoleNotFound on update missing, got %v", err)
	}
	if err := roleRepo.DeleteByID(999999); !errors.Is(err, ErrRoleNotFound) {
		t.Fatalf("expected ErrRoleNotFound on delete missing, got %v", err)
	}

	dup := &domain.Role{Name: "manager-updated", Description: "duplicate"}
	if err := roleRepo.Create(dup, nil); err == nil {
		t.Fatal("expected duplicate role name conflict")
	}
}
