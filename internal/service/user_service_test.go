package service

import (
	"errors"
	"testing"

	"github.com/sandeepkv93/secure-observable-go-backend-starter-kit/internal/domain"
	"github.com/sandeepkv93/secure-observable-go-backend-starter-kit/internal/repository"
)

type stubUserRepository struct {
	findByIDFn func(id uint) (*domain.User, error)
	listFn     func() ([]domain.User, error)
	setRolesFn func(userID uint, roleIDs []uint) error
	addRoleFn  func(userID, roleID uint) error
}

func (s *stubUserRepository) FindByID(id uint) (*domain.User, error) {
	if s.findByIDFn == nil {
		return nil, errors.New("not implemented")
	}
	return s.findByIDFn(id)
}

func (s *stubUserRepository) FindByEmail(_ string) (*domain.User, error) {
	return nil, errors.New("not implemented")
}

func (s *stubUserRepository) Create(_ *domain.User) error { return errors.New("not implemented") }

func (s *stubUserRepository) Update(_ *domain.User) error { return errors.New("not implemented") }

func (s *stubUserRepository) List() ([]domain.User, error) {
	if s.listFn == nil {
		return nil, errors.New("not implemented")
	}
	return s.listFn()
}

func (s *stubUserRepository) ListPaged(_ repository.UserListQuery) (repository.PageResult[domain.User], error) {
	return repository.PageResult[domain.User]{}, errors.New("not implemented")
}

func (s *stubUserRepository) SetRoles(userID uint, roleIDs []uint) error {
	if s.setRolesFn == nil {
		return errors.New("not implemented")
	}
	return s.setRolesFn(userID, roleIDs)
}

func (s *stubUserRepository) AddRole(userID, roleID uint) error {
	if s.addRoleFn == nil {
		return errors.New("not implemented")
	}
	return s.addRoleFn(userID, roleID)
}

func TestUserServiceGetByID(t *testing.T) {
	t.Run("repo error", func(t *testing.T) {
		expected := errors.New("db down")
		repo := &stubUserRepository{
			findByIDFn: func(_ uint) (*domain.User, error) {
				return nil, expected
			},
		}
		svc := NewUserService(repo, NewRBACService())

		u, perms, err := svc.GetByID(1)
		if !errors.Is(err, expected) {
			t.Fatalf("expected %v, got %v", expected, err)
		}
		if u != nil || perms != nil {
			t.Fatal("expected nil user and perms on error")
		}
	})

	t.Run("success derives deduplicated permissions", func(t *testing.T) {
		repo := &stubUserRepository{
			findByIDFn: func(id uint) (*domain.User, error) {
				if id != 7 {
					t.Fatalf("unexpected id %d", id)
				}
				return &domain.User{
					ID:    7,
					Email: "user@example.com",
					Roles: []domain.Role{
						{Name: "viewer", Permissions: []domain.Permission{{Resource: "users", Action: "read"}, {Resource: "roles", Action: "read"}}},
						{Name: "admin", Permissions: []domain.Permission{{Resource: "users", Action: "read"}, {Resource: "roles", Action: "write"}}},
					},
				}, nil
			},
		}
		svc := NewUserService(repo, NewRBACService())

		u, perms, err := svc.GetByID(7)
		if err != nil {
			t.Fatalf("GetByID: %v", err)
		}
		if u == nil || u.ID != 7 {
			t.Fatalf("unexpected user: %+v", u)
		}
		if len(perms) != 3 {
			t.Fatalf("expected 3 deduplicated permissions, got %d (%v)", len(perms), perms)
		}
		assertPermissionSet(t, perms, []string{"users:read", "roles:read", "roles:write"})
	})
}

func TestUserServiceListDelegatesToRepo(t *testing.T) {
	repo := &stubUserRepository{
		listFn: func() ([]domain.User, error) {
			return []domain.User{{ID: 1, Email: "a@example.com"}, {ID: 2, Email: "b@example.com"}}, nil
		},
	}
	svc := NewUserService(repo, NewRBACService())

	users, err := svc.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}
}

func TestUserServiceListError(t *testing.T) {
	expected := errors.New("query failed")
	repo := &stubUserRepository{
		listFn: func() ([]domain.User, error) {
			return nil, expected
		},
	}
	svc := NewUserService(repo, NewRBACService())

	_, err := svc.List()
	if !errors.Is(err, expected) {
		t.Fatalf("expected %v, got %v", expected, err)
	}
}

func TestUserServiceSetRolesDelegatesAndPropagatesErrors(t *testing.T) {
	repo := &stubUserRepository{
		setRolesFn: func(userID uint, roleIDs []uint) error {
			if userID != 7 {
				t.Fatalf("unexpected userID %d", userID)
			}
			if len(roleIDs) != 2 || roleIDs[0] != 1 || roleIDs[1] != 2 {
				t.Fatalf("unexpected roleIDs %v", roleIDs)
			}
			return nil
		},
	}
	svc := NewUserService(repo, NewRBACService())

	if err := svc.SetRoles(7, []uint{1, 2}); err != nil {
		t.Fatalf("SetRoles: %v", err)
	}

	expected := errors.New("replace failed")
	repo.setRolesFn = func(_ uint, _ []uint) error { return expected }
	if err := svc.SetRoles(7, []uint{1, 2}); !errors.Is(err, expected) {
		t.Fatalf("expected %v, got %v", expected, err)
	}
}

func TestUserServiceAddRoleDelegatesAndPropagatesErrors(t *testing.T) {
	repo := &stubUserRepository{
		addRoleFn: func(userID, roleID uint) error {
			if userID != 7 || roleID != 3 {
				t.Fatalf("unexpected args userID=%d roleID=%d", userID, roleID)
			}
			return nil
		},
	}
	svc := NewUserService(repo, NewRBACService())

	if err := svc.AddRole(7, 3); err != nil {
		t.Fatalf("AddRole: %v", err)
	}

	expected := errors.New("append failed")
	repo.addRoleFn = func(_, _ uint) error { return expected }
	if err := svc.AddRole(7, 3); !errors.Is(err, expected) {
		t.Fatalf("expected %v, got %v", expected, err)
	}
}

func assertPermissionSet(t *testing.T, got []string, expected []string) {
	t.Helper()
	set := make(map[string]struct{}, len(got))
	for _, p := range got {
		set[p] = struct{}{}
	}
	for _, want := range expected {
		if _, ok := set[want]; !ok {
			t.Fatalf("missing permission %q in %v", want, got)
		}
	}
}
