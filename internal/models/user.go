package models

import (
	"time"
)

type User struct {
	ID          uint   `gorm:"primaryKey"`
	FirebaseUID string `gorm:"uniqueIndex"`
	PhoneNumber string
	Email       string
	Name        string `json:"name"`
	Address     string
	Role        string `gorm:"default:'buyer'"`
	IsSeller    bool   `gorm:"default:false"`
	FCMToken    string `json:"fcm_token"`
	CreatedAt   time.Time
}
