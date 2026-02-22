package models

import (
	"time"
)

type User struct {
	ID          uint   `gorm:"primaryKey"`
	FirebaseUID string `gorm:"uniqueIndex"`
	PhoneNumber string
	Email       string
	Address     string
	Role        string `gorm:"default:'buyer'"`
	IsSeller    bool   `gorm:"default:false"`
	CreatedAt   time.Time
}
