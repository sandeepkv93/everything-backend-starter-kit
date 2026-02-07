package database

import (
	"go-oauth-rbac-service/internal/domain"

	"gorm.io/gorm"
)

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&domain.User{},
		&domain.Role{},
		&domain.Permission{},
		&domain.UserRole{},
		&domain.RolePermission{},
		&domain.OAuthAccount{},
		&domain.Session{},
	)
}
