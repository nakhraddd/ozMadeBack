package models

import "gorm.io/gorm"

type Seller struct {
	gorm.Model
	UserID                uint   `gorm:"unique;not null"`
	User                  User   `gorm:"foreignKey:UserID"`
	FirstName             string `json:"first_name"`
	LastName              string `json:"last_name"`
	StoreName             string `json:"store_name"`
	City                  string `json:"city"`
	Address               string `json:"address"`
	Description           string `json:"description"`
	Categories            string `json:"categories"`
	PhotoURL              string `json:"photo_url"`
	Status                string `gorm:"default:'pending'"`
	IDCard                string
	Products              []Product `gorm:"foreignKey:SellerID"`
	OrdersCount           int       `json:"orders_count" gorm:"default:0"`
	AverageRating         float64   `json:"average_rating" gorm:"default:0"`
	RatingsCount          int       `json:"ratings_count" gorm:"default:0"`
	ReviewsCount          int       `json:"reviews_count" gorm:"default:0"`
	PickupEnabled         bool      `json:"pickup_enabled"`
	PickupAddress         string    `json:"pickup_address"`
	PickupTime            string    `json:"pickup_time"`
	FreeDeliveryEnabled   bool      `json:"free_delivery_enabled"`
	DeliveryCenterLat     float64   `json:"delivery_center_lat"`
	DeliveryCenterLng     float64   `json:"delivery_center_lng"`
	DeliveryRadiusKm      float64   `json:"delivery_radius_km"`
	DeliveryCenterAddress string    `json:"delivery_center_address"`
	IntercityEnabled      bool      `json:"intercity_enabled"`
}
