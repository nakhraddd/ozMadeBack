package database

import (
	"fmt"
	"ozMadeBack/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Connect(dsn string) {
	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	fmt.Println("Connection Opened to Database")
}

func Migrate() {
	DB.AutoMigrate(
		&models.User{},
		&models.Seller{},
		&models.Product{},
		&models.Chat{},
		&models.Message{},
		&models.Comment{},
		&models.Order{},
		&models.Favorite{},
		&models.Report{},
	)
	fmt.Println("Database Migrated")
}
