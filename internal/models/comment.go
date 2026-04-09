package models

import (
	"time"

	"gorm.io/gorm"
)

type Comment struct {
	gorm.Model // Embedding gorm.Model for ID, CreatedAt, UpdatedAt, DeletedAt
	ProductID  uint
	UserID     uint
	User       User    `gorm:"foreignKey:UserID"`
	Product    Product `gorm:"foreignKey:ProductID"`
	Rating     float64
	Text       string
	CreatedAt  time.Time
}
