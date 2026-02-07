package domain

import "time"

type Permission struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Resource  string    `gorm:"size:64;not null;index:idx_perm_unique,unique" json:"resource"`
	Action    string    `gorm:"size:64;not null;index:idx_perm_unique,unique" json:"action"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type RolePermission struct {
	RoleID       uint      `gorm:"primaryKey"`
	PermissionID uint      `gorm:"primaryKey"`
	CreatedAt    time.Time `json:"created_at"`
}
