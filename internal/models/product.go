package models

import (
	"time"
)

type Product struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	SellerID      uint      `json:"seller_id"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	Type          string    `json:"type"`
	Cost          float64   `json:"cost"`
	Address       string    `json:"address"`
	WhatsAppLink  string    `json:"whats_app_link"`
	ViewCount     int64     `json:"view_count"`
	AverageRating float64   `json:"average_rating" gorm:"default:0"`
	RatingsCount  int       `json:"ratings_count" gorm:"default:0"`
	ReviewsCount  int       `json:"reviews_count" gorm:"default:0"`
	OrdersCount   int       `json:"orders_count" gorm:"default:0"`
	ImageName     string    `json:"image_name"`
	Images        []string  `gorm:"serializer:json" json:"images"`
	Weight        string    `json:"weight"`
	HeightCm      string    `json:"height_cm"`
	WidthCm       string    `json:"width_cm"`
	DepthCm       string    `json:"depth_cm"`
	Composition   string    `json:"composition"`
	YouTubeUrl    string    `json:"you_tube_url"`
	Categories    []string  `gorm:"serializer:json" json:"categories"`
	IsHidden      bool      `json:"is_hidden" gorm:"default:false"`
	CreatedAt     time.Time `json:"created_at"`
	Comments      []Comment `gorm:"foreignKey:ProductID" json:"comments,omitempty"`
	SellerName    string    `gorm:"-" json:"seller_name"`
}
