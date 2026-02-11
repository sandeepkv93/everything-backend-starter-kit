package repository

import (
	"errors"
	"testing"

	"github.com/sandeepkv93/secure-observable-go-backend-starter-kit/internal/domain"
	"gorm.io/gorm"
)

func TestOAuthRepositoryCreateFindAndUniqueness(t *testing.T) {
	db := newRepositoryDBForTest(t)
	repo := NewOAuthRepository(db)

	account := &domain.OAuthAccount{UserID: 1, Provider: "google", ProviderUserID: "provider-1", EmailVerified: true}
	if err := repo.Create(account); err != nil {
		t.Fatalf("create account: %v", err)
	}

	got, err := repo.FindByProvider("google", "provider-1")
	if err != nil {
		t.Fatalf("find by provider: %v", err)
	}
	if got.UserID != 1 || got.Provider != "google" || got.ProviderUserID != "provider-1" {
		t.Fatalf("unexpected account: %+v", got)
	}

	dup := &domain.OAuthAccount{UserID: 2, Provider: "google", ProviderUserID: "provider-1", EmailVerified: true}
	if err := repo.Create(dup); err == nil {
		t.Fatal("expected duplicate provider/provider_user_id conflict")
	}

	_, err = repo.FindByProvider("google", "missing")
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected gorm.ErrRecordNotFound, got %v", err)
	}
}
