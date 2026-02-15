package service

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/sandeepkv93/everything-backend-starter-kit/internal/domain"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestDBIdempotencyStoreCleanupExpiredDeletesOnlyExpiredRows(t *testing.T) {
	store, db := newDBIdempotencyStoreForTest(t)
	now := time.Now().UTC()

	records := []domain.IdempotencyRecord{
		{Scope: "register", IdempotencyKey: "k1", FingerprintHash: "f1", Status: "completed", ExpiresAt: now.Add(-time.Hour)},
		{Scope: "register", IdempotencyKey: "k2", FingerprintHash: "f2", Status: "new", ExpiresAt: now.Add(-2 * time.Minute)},
		{Scope: "register", IdempotencyKey: "k3", FingerprintHash: "f3", Status: "new", ExpiresAt: now.Add(2 * time.Hour)},
	}
	for i := range records {
		if err := db.Create(&records[i]).Error; err != nil {
			t.Fatalf("create record %d: %v", i, err)
		}
	}

	deleted, err := store.CleanupExpired(context.Background(), now, 100)
	if err != nil {
		t.Fatalf("cleanup expired: %v", err)
	}
	if deleted != 2 {
		t.Fatalf("expected 2 deleted rows, got %d", deleted)
	}

	var remaining []domain.IdempotencyRecord
	if err := db.Order("id ASC").Find(&remaining).Error; err != nil {
		t.Fatalf("query remaining: %v", err)
	}
	if len(remaining) != 1 {
		t.Fatalf("expected 1 remaining row, got %d", len(remaining))
	}
	if remaining[0].IdempotencyKey != "k3" {
		t.Fatalf("expected unexpired row to remain, got key=%s", remaining[0].IdempotencyKey)
	}
}

func TestDBIdempotencyStoreCleanupExpiredHonorsBatchSize(t *testing.T) {
	store, db := newDBIdempotencyStoreForTest(t)
	now := time.Now().UTC()

	for i := 0; i < 3; i++ {
		rec := domain.IdempotencyRecord{
			Scope:           "scope",
			IdempotencyKey:  fmt.Sprintf("k-%d", i),
			FingerprintHash: fmt.Sprintf("f-%d", i),
			Status:          "completed",
			ExpiresAt:       now.Add(-time.Minute),
		}
		if err := db.Create(&rec).Error; err != nil {
			t.Fatalf("create expired record %d: %v", i, err)
		}
	}

	deleted, err := store.CleanupExpired(context.Background(), now, 1)
	if err != nil {
		t.Fatalf("cleanup expired: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("expected 1 deleted row with batch=1, got %d", deleted)
	}

	var count int64
	if err := db.Model(&domain.IdempotencyRecord{}).Count(&count).Error; err != nil {
		t.Fatalf("count remaining: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 remaining rows, got %d", count)
	}
}

func newDBIdempotencyStoreForTest(t *testing.T) (*DBIdempotencyStore, *gorm.DB) {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&domain.IdempotencyRecord{}); err != nil {
		t.Fatalf("migrate idempotency record: %v", err)
	}
	return NewDBIdempotencyStore(db), db
}
