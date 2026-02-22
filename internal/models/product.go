package models

import (
	"time"
)

type Product struct {
	ID            uint `gorm:"primaryKey"`
	SellerID      uint
	Title         string
	Description   string
	Type          string
	Cost          float64
	Address       string
	WhatsAppLink  string
	ViewCount     int64
	AverageRating float64
	ImageName     string
	CreatedAt     time.Time
	Comments      []Comment
}
