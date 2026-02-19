package database

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
	"ozMadeBack/config"
	"ozMadeBack/internal/models"
)

var DB *gorm.DB

func InitDatabase() {
	var err error
	dsn := config.GetEnv("DB_DSN")
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	if err := DB.AutoMigrate(&models.User{}); err != nil {
		log.Fatalf("Failed to auto-migrate database: %v", err)
	}
}
