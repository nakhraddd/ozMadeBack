package models

import (
	"time"
)

type User struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	FirebaseUID string    `gorm:"uniqueIndex" json:"firebase_uid"`
	PhoneNumber string    `json:"phone_number"`
	PhotoUrl    string    `json:"photo_url"`
	Name        string    `json:"name"`
	Address     string    `json:"address"`
	Role        string    `gorm:"default:'buyer'" json:"role"`
	IsSeller    bool      `gorm:"default:false" json:"is_seller"`
	FCMToken    string    `json:"fcm_token"`
	CreatedAt   time.Time `json:"created_at"`
}
