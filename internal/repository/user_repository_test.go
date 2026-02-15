package repository

import (
	"testing"

	"github.com/sandeepkv93/everything-backend-starter-kit/internal/domain"
)

func TestUserRepositoryListPagedFiltersSortAndRoleAssociations(t *testing.T) {
	db := newRepositoryDBForTest(t)
	userRepo := NewUserRepository(db)
	roleRepo := NewRoleRepository(db)

	permRead := &domain.Permission{Resource: "users", Action: "read"}
	if err := db.Create(permRead).Error; err != nil {
		t.Fatalf("create permission: %v", err)
	}
	adminRole := &domain.Role{Name: "admin"}
	userRole := &domain.Role{Name: "user"}
	if err := roleRepo.Create(adminRole, []uint{permRead.ID}); err != nil {
		t.Fatalf("create admin role: %v", err)
	}
	if err := roleRepo.Create(userRole, nil); err != nil {
		t.Fatalf("create user role: %v", err)
	}

	u1 := &domain.User{Email: "alice@example.com", Name: "Alice", Status: "active"}
	u2 := &domain.User{Email: "bob@example.com", Name: "Bob", Status: "disabled"}
	u3 := &domain.User{Email: "charlie@example.com", Name: "Charlie", Status: "active"}
	for _, u := range []*domain.User{u1, u2, u3} {
		if err := userRepo.Create(u); err != nil {
			t.Fatalf("create user %s: %v", u.Email, err)
		}
	}
	if err := userRepo.AddRole(u1.ID, adminRole.ID); err != nil {
		t.Fatalf("add admin role: %v", err)
	}
	if err := userRepo.AddRole(u2.ID, userRole.ID); err != nil {
		t.Fatalf("add user role u2: %v", err)
	}
	if err := userRepo.AddRole(u3.ID, userRole.ID); err != nil {
		t.Fatalf("add user role u3: %v", err)
	}

	page, err := userRepo.ListPaged(UserListQuery{
		PageRequest: PageRequest{Page: 1, PageSize: 2},
		SortBy:      "email",
		SortOrder:   "asc",
		Email:       "a",
		Status:      "active",
	})
	if err != nil {
		t.Fatalf("list paged with email/status filters: %v", err)
	}
	if page.Total != 1 || len(page.Items) != 1 || page.Items[0].Email != "alice@example.com" {
		t.Fatalf("unexpected email/status filtered page: %+v", page)
	}

	rolePage, err := userRepo.ListPaged(UserListQuery{
		PageRequest: PageRequest{Page: 1, PageSize: 10},
		SortBy:      "created_at",
		SortOrder:   "desc",
		Role:        "user",
	})
	if err != nil {
		t.Fatalf("list paged by role: %v", err)
	}
	if rolePage.Total != 2 || len(rolePage.Items) != 2 {
		t.Fatalf("expected 2 users with role=user, got total=%d items=%d", rolePage.Total, len(rolePage.Items))
	}

	if err := userRepo.SetRoles(u1.ID, []uint{userRole.ID}); err != nil {
		t.Fatalf("set roles replace: %v", err)
	}
	updated, err := userRepo.FindByID(u1.ID)
	if err != nil {
		t.Fatalf("find user after set roles: %v", err)
	}
	if len(updated.Roles) != 1 || updated.Roles[0].Name != "user" {
		t.Fatalf("expected roles replaced to [user], got %+v", updated.Roles)
	}
}
