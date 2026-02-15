package repository

import (
	"fmt"
	"strings"
	"testing"

	"github.com/sandeepkv93/everything-backend-starter-kit/internal/domain"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newRepositoryDBForTest(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&domain.Permission{},
		&domain.Role{},
		&domain.User{},
		&domain.LocalCredential{},
		&domain.VerificationToken{},
		&domain.OAuthAccount{},
		&domain.Session{},
	); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	return db
}
