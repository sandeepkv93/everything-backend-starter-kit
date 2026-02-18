package domain

import "time"

type FeatureFlag struct {
	ID          uint              `gorm:"primaryKey" json:"id"`
	Key         string            `gorm:"uniqueIndex;size:128;not null" json:"key"`
	Description string            `gorm:"size:512" json:"description"`
	Enabled     bool              `gorm:"not null;default:false" json:"enabled"`
	Rules       []FeatureFlagRule `gorm:"foreignKey:FeatureFlagID;constraint:OnDelete:CASCADE" json:"rules,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

type FeatureFlagRule struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	FeatureFlagID uint      `gorm:"not null;index" json:"feature_flag_id"`
	Type          string    `gorm:"size:32;not null;index" json:"type"`
	MatchValue    string    `gorm:"size:255" json:"match_value"`
	Percentage    int       `gorm:"not null;default:0" json:"percentage"`
	Enabled       bool      `gorm:"not null;default:false" json:"enabled"`
	Priority      int       `gorm:"not null;default:100" json:"priority"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
