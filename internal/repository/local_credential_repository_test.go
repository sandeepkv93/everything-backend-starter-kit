package repository

import (
	"strings"
	"testing"
	"time"

	"github.com/sandeepkv93/everything-backend-starter-kit/internal/domain"
)

func TestLocalCredentialRepositoryFindByEmailJoinAndUpdates(t *testing.T) {
	db := newRepositoryDBForTest(t)
	repo := NewLocalCredentialRepository(db)

	user := &domain.User{Email: "test@example.com", Name: "Test", Status: "active"}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	cred := &domain.LocalCredential{UserID: user.ID, PasswordHash: "hash-1", EmailVerified: false}
	if err := repo.Create(cred); err != nil {
		t.Fatalf("create credential: %v", err)
	}

	got, err := repo.FindByEmail("  TEST@example.com ")
	if err != nil {
		t.Fatalf("find by email: %v", err)
	}
	if got.UserID != user.ID || got.PasswordHash != "hash-1" {
		t.Fatalf("unexpected credential: %+v", got)
	}

	if err := repo.UpdatePassword(user.ID, "hash-2"); err != nil {
		t.Fatalf("update password: %v", err)
	}
	updated, err := repo.FindByUserID(user.ID)
	if err != nil {
		t.Fatalf("find by user id: %v", err)
	}
	if updated.PasswordHash != "hash-2" {
		t.Fatalf("expected updated hash, got %q", updated.PasswordHash)
	}

	before := time.Now().UTC().Add(-time.Second)
	if err := repo.MarkEmailVerified(user.ID); err != nil {
		t.Fatalf("mark verified: %v", err)
	}
	verified, err := repo.FindByUserID(user.ID)
	if err != nil {
		t.Fatalf("find verified credential: %v", err)
	}
	if !verified.EmailVerified {
		t.Fatal("expected EmailVerified=true")
	}
	if verified.EmailVerifiedAt == nil || verified.EmailVerifiedAt.Before(before) {
		t.Fatalf("expected EmailVerifiedAt to be set, got %v", verified.EmailVerifiedAt)
	}

	_, err = repo.FindByEmail("missing@example.com")
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "record not found") {
		t.Fatalf("expected record not found error, got %v", err)
	}
}
