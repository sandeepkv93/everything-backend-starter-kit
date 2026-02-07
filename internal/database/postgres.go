package database

import (
	"go-oauth-rbac-service/internal/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Open(cfg *config.Config) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
}
