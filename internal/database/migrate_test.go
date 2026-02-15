package database

import (
	"testing"

	"github.com/sandeepkv93/everything-backend-starter-kit/internal/domain"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newSQLiteDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	return db
}

func TestMigrateSuccessCreatesTables(t *testing.T) {
	db := newSQLiteDB(t)
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	if !db.Migrator().HasTable(&domain.User{}) {
		t.Fatal("expected users table")
	}
	if !db.Migrator().HasTable(&domain.Role{}) {
		t.Fatal("expected roles table")
	}
}

func TestMigrateFailureWhenDBClosed(t *testing.T) {
	db := newSQLiteDB(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close sql db: %v", err)
	}

	if err := Migrate(db); err == nil {
		t.Fatal("expected migrate error on closed database")
	}
}
