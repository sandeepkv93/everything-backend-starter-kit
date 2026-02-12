package service

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sandeepkv93/secure-observable-go-backend-starter-kit/internal/domain"
	"github.com/sandeepkv93/secure-observable-go-backend-starter-kit/internal/repository"
	"github.com/sandeepkv93/secure-observable-go-backend-starter-kit/internal/security"
)

type stubSessionRepository struct {
	listActiveByUserIDFn         func(userID uint) ([]domain.Session, error)
	findActiveByTokenIDForUserFn func(userID uint, tokenID string) (*domain.Session, error)
	findByHashFn                 func(hash string) (*domain.Session, error)
	revokeByIDForUserFn          func(userID, sessionID uint, reason string) (bool, error)
	revokeOthersByUserFn         func(userID, keepSessionID uint, reason string) (int64, error)
}

func (s *stubSessionRepository) Create(_ *domain.Session) error { return errors.New("not implemented") }
func (s *stubSessionRepository) FindByHash(hash string) (*domain.Session, error) {
	if s.findByHashFn == nil {
		return nil, errors.New("not implemented")
	}
	return s.findByHashFn(hash)
}
func (s *stubSessionRepository) FindActiveByTokenIDForUser(userID uint, tokenID string) (*domain.Session, error) {
	if s.findActiveByTokenIDForUserFn == nil {
		return nil, errors.New("not implemented")
	}
	return s.findActiveByTokenIDForUserFn(userID, tokenID)
}
func (s *stubSessionRepository) FindByIDForUser(_, _ uint) (*domain.Session, error) {
	return nil, errors.New("not implemented")
}
func (s *stubSessionRepository) ListActiveByUserID(userID uint) ([]domain.Session, error) {
	if s.listActiveByUserIDFn == nil {
		return nil, errors.New("not implemented")
	}
	return s.listActiveByUserIDFn(userID)
}
func (s *stubSessionRepository) RotateSession(_ string, _ *domain.Session) (*domain.Session, error) {
	return nil, errors.New("not implemented")
}
func (s *stubSessionRepository) UpdateTokenLineageByHash(_, _, _ string) error {
	return errors.New("not implemented")
}
func (s *stubSessionRepository) MarkReuseDetectedByHash(_ string) error {
	return errors.New("not implemented")
}
func (s *stubSessionRepository) RevokeByHash(_, _ string) error { return errors.New("not implemented") }
func (s *stubSessionRepository) RevokeByIDForUser(userID, sessionID uint, reason string) (bool, error) {
	if s.revokeByIDForUserFn == nil {
		return false, errors.New("not implemented")
	}
	return s.revokeByIDForUserFn(userID, sessionID, reason)
}
func (s *stubSessionRepository) RevokeOthersByUser(userID, keepSessionID uint, reason string) (int64, error) {
	if s.revokeOthersByUserFn == nil {
		return 0, errors.New("not implemented")
	}
	return s.revokeOthersByUserFn(userID, keepSessionID, reason)
}
func (s *stubSessionRepository) RevokeByFamilyID(_, _ string) (int64, error) {
	return 0, errors.New("not implemented")
}
func (s *stubSessionRepository) RevokeByUserID(_ uint, _ string) error {
	return errors.New("not implemented")
}
func (s *stubSessionRepository) CleanupExpired() (int64, error) {
	return 0, errors.New("not implemented")
}

func TestSessionServiceListActiveSessions(t *testing.T) {
	now := time.Now().UTC()
	revoked := now.Add(-time.Minute)

	repo := &stubSessionRepository{
		listActiveByUserIDFn: func(userID uint) ([]domain.Session, error) {
			if userID != 42 {
				t.Fatalf("unexpected userID: %d", userID)
			}
			return []domain.Session{
				{ID: 10, CreatedAt: now.Add(-2 * time.Hour), ExpiresAt: now.Add(time.Hour), UserAgent: "ua1", IP: "1.1.1.1"},
				{ID: 11, CreatedAt: now.Add(-time.Hour), ExpiresAt: now.Add(2 * time.Hour), RevokedAt: &revoked, UserAgent: "ua2", IP: "2.2.2.2"},
			}, nil
		},
	}
	svc := NewSessionService(repo, "pepper")

	views, err := svc.ListActiveSessions(42, 11)
	if err != nil {
		t.Fatalf("ListActiveSessions: %v", err)
	}
	if len(views) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(views))
	}
	if views[0].IsCurrent {
		t.Fatal("expected first session not current")
	}
	if !views[1].IsCurrent {
		t.Fatal("expected second session current")
	}
	if views[1].RevokedAt == nil {
		t.Fatal("expected revoked_at to be mapped")
	}
}

func TestSessionServiceListActiveSessionsRepoError(t *testing.T) {
	expected := errors.New("db unavailable")
	repo := &stubSessionRepository{
		listActiveByUserIDFn: func(_ uint) ([]domain.Session, error) {
			return nil, expected
		},
	}
	svc := NewSessionService(repo, "pepper")

	_, err := svc.ListActiveSessions(1, 0)
	if !errors.Is(err, expected) {
		t.Fatalf("expected %v, got %v", expected, err)
	}
}

func TestSessionServiceResolveCurrentSessionID(t *testing.T) {
	t.Run("claims token success", func(t *testing.T) {
		repo := &stubSessionRepository{
			findActiveByTokenIDForUserFn: func(userID uint, tokenID string) (*domain.Session, error) {
				if userID != 7 || tokenID != "token-123" {
					t.Fatalf("unexpected find args userID=%d tokenID=%q", userID, tokenID)
				}
				return &domain.Session{ID: 77}, nil
			},
		}
		svc := NewSessionService(repo, "pepper")
		req := httptest.NewRequest("GET", "/", nil)

		id, err := svc.ResolveCurrentSessionID(req, &security.Claims{
			RegisteredClaims: jwt.RegisteredClaims{ID: "token-123"},
		}, 7)
		if err != nil {
			t.Fatalf("ResolveCurrentSessionID: %v", err)
		}
		if id != 77 {
			t.Fatalf("expected session ID 77, got %d", id)
		}
	})

	t.Run("claims lookup unexpected error", func(t *testing.T) {
		expected := errors.New("db down")
		repo := &stubSessionRepository{
			findActiveByTokenIDForUserFn: func(_ uint, _ string) (*domain.Session, error) {
				return nil, expected
			},
		}
		svc := NewSessionService(repo, "pepper")
		req := httptest.NewRequest("GET", "/", nil)

		_, err := svc.ResolveCurrentSessionID(req, &security.Claims{
			RegisteredClaims: jwt.RegisteredClaims{ID: "token-123"},
		}, 7)
		if !errors.Is(err, expected) {
			t.Fatalf("expected %v, got %v", expected, err)
		}
	})

	t.Run("claims not found falls back to cookie hash", func(t *testing.T) {
		refreshToken := "refresh-token"
		pepper := "p3pp3r"
		expectedHash := security.HashRefreshToken(refreshToken, pepper)

		repo := &stubSessionRepository{
			findActiveByTokenIDForUserFn: func(_ uint, _ string) (*domain.Session, error) {
				return nil, repository.ErrSessionNotFound
			},
			findByHashFn: func(hash string) (*domain.Session, error) {
				if hash != expectedHash {
					t.Fatalf("unexpected hash %q", hash)
				}
				return &domain.Session{ID: 42, UserID: 7, ExpiresAt: time.Now().Add(time.Hour)}, nil
			},
		}
		svc := NewSessionService(repo, pepper)
		req := httptest.NewRequest("GET", "/", nil)
		req.AddCookie(&http.Cookie{Name: "refresh_token", Value: refreshToken})

		id, err := svc.ResolveCurrentSessionID(req, &security.Claims{
			RegisteredClaims: jwt.RegisteredClaims{ID: "token-123"},
		}, 7)
		if err != nil {
			t.Fatalf("ResolveCurrentSessionID: %v", err)
		}
		if id != 42 {
			t.Fatalf("expected session ID 42, got %d", id)
		}
	})

	t.Run("missing cookie returns not found", func(t *testing.T) {
		repo := &stubSessionRepository{}
		svc := NewSessionService(repo, "pepper")
		req := httptest.NewRequest("GET", "/", nil)

		_, err := svc.ResolveCurrentSessionID(req, nil, 7)
		if !errors.Is(err, repository.ErrSessionNotFound) {
			t.Fatalf("expected ErrSessionNotFound, got %v", err)
		}
	})

	t.Run("fallback hash repo error", func(t *testing.T) {
		expected := errors.New("redis unavailable")
		repo := &stubSessionRepository{
			findByHashFn: func(_ string) (*domain.Session, error) {
				return nil, expected
			},
		}
		svc := NewSessionService(repo, "pepper")
		req := httptest.NewRequest("GET", "/", nil)
		req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "refresh-token"})

		_, err := svc.ResolveCurrentSessionID(req, nil, 7)
		if !errors.Is(err, expected) {
			t.Fatalf("expected %v, got %v", expected, err)
		}
	})

	t.Run("fallback session rejected on mismatched user revoked or expired", func(t *testing.T) {
		cases := []struct {
			name    string
			session *domain.Session
		}{
			{name: "mismatched user", session: &domain.Session{ID: 1, UserID: 99, ExpiresAt: time.Now().Add(time.Hour)}},
			{name: "revoked", session: &domain.Session{ID: 1, UserID: 7, ExpiresAt: time.Now().Add(time.Hour), RevokedAt: ptrTime(time.Now())}},
			{name: "expired", session: &domain.Session{ID: 1, UserID: 7, ExpiresAt: time.Now().Add(-time.Minute)}},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				repo := &stubSessionRepository{
					findByHashFn: func(_ string) (*domain.Session, error) {
						return tc.session, nil
					},
				}
				svc := NewSessionService(repo, "pepper")
				req := httptest.NewRequest("GET", "/", nil)
				req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "refresh-token"})

				_, err := svc.ResolveCurrentSessionID(req, nil, 7)
				if !errors.Is(err, repository.ErrSessionNotFound) {
					t.Fatalf("expected ErrSessionNotFound, got %v", err)
				}
			})
		}
	})
}

func TestSessionServiceRevokeSession(t *testing.T) {
	t.Run("repo error", func(t *testing.T) {
		expected := errors.New("update failed")
		repo := &stubSessionRepository{
			revokeByIDForUserFn: func(_, _ uint, _ string) (bool, error) { return false, expected },
		}
		svc := NewSessionService(repo, "pepper")

		_, err := svc.RevokeSession(1, 2)
		if !errors.Is(err, expected) {
			t.Fatalf("expected %v, got %v", expected, err)
		}
	})

	t.Run("already revoked", func(t *testing.T) {
		repo := &stubSessionRepository{
			revokeByIDForUserFn: func(_, _ uint, reason string) (bool, error) {
				if reason != "user_session_revoked" {
					t.Fatalf("unexpected reason %q", reason)
				}
				return false, nil
			},
		}
		svc := NewSessionService(repo, "pepper")

		status, err := svc.RevokeSession(1, 2)
		if err != nil {
			t.Fatalf("RevokeSession: %v", err)
		}
		if status != "already_revoked" {
			t.Fatalf("expected already_revoked, got %q", status)
		}
	})

	t.Run("revoked", func(t *testing.T) {
		repo := &stubSessionRepository{
			revokeByIDForUserFn: func(_, _ uint, _ string) (bool, error) { return true, nil },
		}
		svc := NewSessionService(repo, "pepper")

		status, err := svc.RevokeSession(1, 2)
		if err != nil {
			t.Fatalf("RevokeSession: %v", err)
		}
		if status != "revoked" {
			t.Fatalf("expected revoked, got %q", status)
		}
	})
}

func TestSessionServiceRevokeOtherSessions(t *testing.T) {
	repo := &stubSessionRepository{
		revokeOthersByUserFn: func(userID, keepSessionID uint, reason string) (int64, error) {
			if userID != 9 || keepSessionID != 3 {
				t.Fatalf("unexpected args userID=%d keepSessionID=%d", userID, keepSessionID)
			}
			if reason != "user_revoke_others" {
				t.Fatalf("unexpected reason %q", reason)
			}
			return 4, nil
		},
	}
	svc := NewSessionService(repo, "pepper")

	n, err := svc.RevokeOtherSessions(9, 3)
	if err != nil {
		t.Fatalf("RevokeOtherSessions: %v", err)
	}
	if n != 4 {
		t.Fatalf("expected 4 revocations, got %d", n)
	}
}

func ptrTime(t time.Time) *time.Time { return &t }
