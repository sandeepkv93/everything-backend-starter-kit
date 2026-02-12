package database

import (
	"errors"
	"testing"

	"github.com/sandeepkv93/secure-observable-go-backend-starter-kit/internal/domain"
	"gorm.io/gorm"
)

func TestSeedSyncCreatesDataAndNoopOnSecondRun(t *testing.T) {
	db := newSQLiteDB(t)
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	report1, err := SeedSync(db, "")
	if err != nil {
		t.Fatalf("seed sync first run: %v", err)
	}
	if report1.Noop {
		t.Fatalf("expected first seed run to perform changes: %+v", report1)
	}
	if report1.CreatedPermissions == 0 || report1.CreatedRoles == 0 {
		t.Fatalf("expected created permissions and roles: %+v", report1)
	}

	report2, err := SeedSync(db, "")
	if err != nil {
		t.Fatalf("seed sync second run: %v", err)
	}
	if !report2.Noop {
		t.Fatalf("expected noop on second seed run: %+v", report2)
	}
}

func TestSeedSyncFailureWhenDBClosed(t *testing.T) {
	db := newSQLiteDB(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close sql db: %v", err)
	}

	if _, err := SeedSync(db, ""); err == nil {
		t.Fatal("expected seed sync error on closed database")
	}
}

func TestVerifyLocalEmailValidationAndBehavior(t *testing.T) {
	db := newSQLiteDB(t)
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	if err := VerifyLocalEmail(db, ""); err == nil {
		t.Fatal("expected email required error")
	}

	if err := VerifyLocalEmail(db, "missing@example.com"); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected record not found for missing user, got %v", err)
	}

	u := domain.User{Email: "user@example.com", Name: "User", Status: "active"}
	if err := db.Create(&u).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	cred := domain.LocalCredential{UserID: u.ID, PasswordHash: "hash", EmailVerified: false}
	if err := db.Create(&cred).Error; err != nil {
		t.Fatalf("create local credential: %v", err)
	}

	if err := VerifyLocalEmail(db, "  USER@example.com "); err != nil {
		t.Fatalf("verify local email: %v", err)
	}

	var refreshed domain.LocalCredential
	if err := db.Where("user_id = ?", u.ID).First(&refreshed).Error; err != nil {
		t.Fatalf("reload credential: %v", err)
	}
	if !refreshed.EmailVerified || refreshed.EmailVerifiedAt == nil {
		t.Fatalf("expected verified credential with timestamp, got %+v", refreshed)
	}
}
