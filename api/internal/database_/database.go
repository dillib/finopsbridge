package database

import (
	"finopsbridge/api/internal/models_"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Initialize(databaseURL string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}

	// Auto-migrate models
	if err := db.AutoMigrate(
		&models_.User{},
		&models_.Organization{},
		&models_.CloudProvider{},
		&models_.Policy{},
		&models_.PolicyViolation{},
		&models_.ActivityLog{},
		&models_.WaitlistEntry{},
		&models_.Webhook{},
	); err != nil {
		return nil, err
	}

	return db, nil
}

