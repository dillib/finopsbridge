package database

import (
	models "finopsbridge/api/internal/models_"

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
		&models.User{},
		&models.Organization{},
		&models.CloudProvider{},
		&models.Policy{},
		&models.PolicyViolation{},
		&models.ActivityLog{},
		&models.WaitlistEntry{},
		&models.Webhook{},
		&models.PolicyCategory{},
		&models.PolicyTemplate{},
		&models.PolicyRecommendation{},
		&models.PolicyAdoptionMetrics{},
	); err != nil {
		return nil, err
	}

	return db, nil
}

