package repository

import (
	"errors"
	"testing"

	"github.com/sandeepkv93/everything-backend-starter-kit/internal/domain"
)

func TestPermissionRepositoryListPagedFindByPairsAndConflicts(t *testing.T) {
	db := newRepositoryDBForTest(t)
	repo := NewPermissionRepository(db)

	seed := []domain.Permission{
		{Resource: "users", Action: "read"},
		{Resource: "users", Action: "write"},
		{Resource: "roles", Action: "read"},
		{Resource: "reports", Action: "read"},
	}
	for i := range seed {
		if err := repo.Create(&seed[i]); err != nil {
			t.Fatalf("create seed permission: %v", err)
		}
	}

	page, err := repo.ListPaged(PageRequest{Page: 1, PageSize: 2}, "resource", "asc", "u", "")
	if err != nil {
		t.Fatalf("list paged: %v", err)
	}
	if page.Total != 2 || len(page.Items) != 2 || page.Items[0].Resource != "users" {
		t.Fatalf("unexpected filtered page result: %+v", page)
	}

	pairs := [][2]string{{"users", "read"}, {"users", "write"}, {"users", "read"}, {"missing", "x"}}
	got, err := repo.FindByPairs(pairs)
	if err != nil {
		t.Fatalf("find by pairs: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 matching unique permissions, got %d (%+v)", len(got), got)
	}

	dup := &domain.Permission{Resource: "users", Action: "read"}
	if err := repo.Create(dup); err == nil {
		t.Fatal("expected unique conflict creating duplicate permission")
	}

	if err := repo.Update(&domain.Permission{ID: 999999, Resource: "x", Action: "y"}); !errors.Is(err, ErrPermissionNotFound) {
		t.Fatalf("expected ErrPermissionNotFound on update missing, got %v", err)
	}
	if err := repo.DeleteByID(999999); !errors.Is(err, ErrPermissionNotFound) {
		t.Fatalf("expected ErrPermissionNotFound on delete missing, got %v", err)
	}
}
