package repository

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/sandeepkv93/secure-observable-go-backend-starter-kit/internal/domain"
)

func TestVerificationTokenRepositoryInvalidateFindConsume(t *testing.T) {
	db := newRepositoryDBForTest(t)
	repo := NewVerificationTokenRepository(db)
	now := time.Now().UTC()

	active := &domain.VerificationToken{UserID: 11, TokenHash: "hash-active", Purpose: "email_verify", ExpiresAt: now.Add(30 * time.Minute)}
	expired := &domain.VerificationToken{UserID: 11, TokenHash: "hash-expired", Purpose: "email_verify", ExpiresAt: now.Add(-5 * time.Minute)}
	otherPurpose := &domain.VerificationToken{UserID: 11, TokenHash: "hash-other", Purpose: "password_reset", ExpiresAt: now.Add(30 * time.Minute)}
	for _, tok := range []*domain.VerificationToken{active, expired, otherPurpose} {
		if err := repo.Create(tok); err != nil {
			t.Fatalf("create token %s: %v", tok.TokenHash, err)
		}
	}

	if err := repo.InvalidateActiveByUserPurpose(11, "email_verify", now); err != nil {
		t.Fatalf("invalidate active tokens: %v", err)
	}

	_, err := repo.FindActiveByHashPurpose("hash-active", "email_verify", now)
	if !errors.Is(err, ErrVerificationTokenNotFound) {
		t.Fatalf("expected invalidated token not found, got %v", err)
	}
	stillActive, err := repo.FindActiveByHashPurpose("hash-other", "password_reset", now)
	if err != nil {
		t.Fatalf("expected other-purpose token still active: %v", err)
	}
	if stillActive.TokenHash != "hash-other" {
		t.Fatalf("unexpected token returned: %+v", stillActive)
	}
}

func TestVerificationTokenRepositoryConsumeIdempotencyAndConcurrency(t *testing.T) {
	db := newRepositoryDBForTest(t)
	repo := NewVerificationTokenRepository(db)
	now := time.Now().UTC()
	token := &domain.VerificationToken{UserID: 21, TokenHash: "hash-consume", Purpose: "password_reset", ExpiresAt: now.Add(time.Hour)}
	if err := repo.Create(token); err != nil {
		t.Fatalf("create token: %v", err)
	}

	if err := repo.Consume(token.ID, token.UserID, now); err != nil {
		t.Fatalf("first consume: %v", err)
	}
	if err := repo.Consume(token.ID, token.UserID, now.Add(time.Second)); !errors.Is(err, ErrVerificationTokenNotFound) {
		t.Fatalf("expected second consume to return ErrVerificationTokenNotFound, got %v", err)
	}

	concurrent := &domain.VerificationToken{UserID: 22, TokenHash: "hash-concurrent", Purpose: "email_verify", ExpiresAt: now.Add(time.Hour)}
	if err := repo.Create(concurrent); err != nil {
		t.Fatalf("create concurrent token: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	errs := make([]error, 2)
	for i := 0; i < 2; i++ {
		idx := i
		go func() {
			defer wg.Done()
			errs[idx] = repo.Consume(concurrent.ID, concurrent.UserID, now.Add(2*time.Second))
		}()
	}
	wg.Wait()

	success := 0
	notFound := 0
	for _, err := range errs {
		switch {
		case err == nil:
			success++
		case errors.Is(err, ErrVerificationTokenNotFound):
			notFound++
		default:
			t.Fatalf("unexpected consume error: %v", err)
		}
	}
	if success != 1 || notFound != 1 {
		t.Fatalf("expected one success and one not-found, got success=%d notFound=%d errs=%v", success, notFound, errs)
	}
}
