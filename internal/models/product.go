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
	AverageRating float64 `json:"average_rating" gorm:"default:0"`
	RatingsCount  int     `json:"ratings_count" gorm:"default:0"`
	ReviewsCount  int     `json:"reviews_count" gorm:"default:0"`
	OrdersCount   int     `json:"orders_count" gorm:"default:0"`
	ImageName     string
	Images        []string `gorm:"serializer:json"`
	Weight        string
	HeightCm      string
	WidthCm       string
	DepthCm       string
	Composition   string
	YouTubeUrl    string
	Categories    []string `gorm:"serializer:json"`
	CreatedAt     time.Time
	Comments      []Comment `gorm:"foreignKey:ProductID"`
	SellerName    string    `gorm:"-"`
}
