package domain

import "time"

type OAuthAccount struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	UserID         uint      `gorm:"index;not null" json:"user_id"`
	Provider       string    `gorm:"size:32;index:idx_provider_uid,unique;not null" json:"provider"`
	ProviderUserID string    `gorm:"size:255;index:idx_provider_uid,unique;not null" json:"provider_user_id"`
	EmailVerified  bool      `gorm:"not null;default:false" json:"email_verified"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
