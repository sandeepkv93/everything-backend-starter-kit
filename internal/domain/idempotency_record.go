package domain

import "time"

type IdempotencyRecord struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	Scope           string    `gorm:"size:256;not null;uniqueIndex:idx_idempotency_scope_key" json:"-"`
	IdempotencyKey  string    `gorm:"size:128;not null;uniqueIndex:idx_idempotency_scope_key" json:"-"`
	FingerprintHash string    `gorm:"size:128;not null" json:"-"`
	Status          string    `gorm:"size:32;not null;index" json:"-"`
	ResponseStatus  int       `json:"-"`
	ResponseBody    []byte    `gorm:"type:bytes" json:"-"`
	ContentType     string    `gorm:"size:128" json:"-"`
	ExpiresAt       time.Time `gorm:"index;not null" json:"-"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
