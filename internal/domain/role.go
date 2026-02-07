package domain

import "time"

type Role struct {
	ID          uint         `gorm:"primaryKey" json:"id"`
	Name        string       `gorm:"uniqueIndex;size:64;not null" json:"name"`
	Description string       `gorm:"size:255" json:"description"`
	Permissions []Permission `gorm:"many2many:role_permissions" json:"permissions,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

type UserRole struct {
	UserID    uint      `gorm:"primaryKey"`
	RoleID    uint      `gorm:"primaryKey"`
	CreatedAt time.Time `json:"created_at"`
}
