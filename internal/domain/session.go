package domain

import "time"

type Session struct {
	ID               uint       `gorm:"primaryKey" json:"id"`
	UserID           uint       `gorm:"index;not null" json:"user_id"`
	RefreshTokenHash string     `gorm:"size:128;uniqueIndex;not null" json:"-"`
	UserAgent        string     `gorm:"size:512" json:"user_agent"`
	IP               string     `gorm:"size:64" json:"ip"`
	ExpiresAt        time.Time  `gorm:"index;not null" json:"expires_at"`
	RevokedAt        *time.Time `gorm:"index" json:"revoked_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}
