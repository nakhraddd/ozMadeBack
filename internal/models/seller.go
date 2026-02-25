package models

import "gorm.io/gorm"

type Seller struct {
	gorm.Model
	UserID                uint   `gorm:"unique;not null"`
	User                  User   `gorm:"foreignKey:UserID"`
	Status                string `gorm:"default:'pending'"`
	IDCard                string
	Products              []Product `gorm:"foreignKey:SellerID"`
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
