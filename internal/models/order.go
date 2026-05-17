package models

import (
	"time"
)

// Order Status Constants
const (
	StatusPendingSeller     = "PENDING_SELLER"
	StatusConfirmed         = "CONFIRMED"
	StatusReadyOrShipped    = "READY_OR_SHIPPED"
	StatusCompleted         = "COMPLETED"
	StatusCancelledByBuyer  = "CANCELLED_BY_BUYER"
	StatusCancelledBySeller = "CANCELLED_BY_SELLER"
	StatusExpired           = "EXPIRED"
)

// Delivery Type Constants
const (
	DeliveryTypePickup     = "PICKUP"
	DeliveryTypeMyDelivery = "MY_DELIVERY"
	DeliveryTypeIntercity  = "INTERCITY"
)

// AddressDetails struct for intercity delivery
type AddressDetails struct {
	City        string  `json:"city"`
	FullAddress string  `json:"fullAddress"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
}

// PackageDetails struct for intercity delivery
type PackageDetails struct {
	WeightGrams int `json:"weightGrams"`
	HeightCm    int `json:"heightCm"`
	WidthCm     int `json:"widthCm"`
	DepthCm     int `json:"depthCm"`
}

// IntercityDeliveryDetails struct to store CDEK-related information
type IntercityDeliveryDetails struct {
	Provider          string         `json:"provider"`
	Price             float64        `json:"price"`
	Currency          string         `json:"currency"`
	MinDays           int            `json:"minDays"`
	MaxDays           int            `json:"maxDays"`
	EstimatedDateFrom string         `json:"estimatedDateFrom"`
	EstimatedDateTo   string         `json:"estimatedDateTo"`
	FromAddress       AddressDetails `json:"fromAddress"`
	ToAddress         AddressDetails `json:"toAddress"`
	Package           PackageDetails `json:"package"`
	ReceiverName      string         `json:"receiverName"`
	ReceiverPhone     string         `json:"receiverPhone"`
	ReceiverAddress   string         `json:"receiverAddress"`
}

type Order struct {
	ID                  uint      `gorm:"primaryKey"`
	UserID              uint      `json:"user_id"`
	ProductID           uint      `json:"product_id"`
	Quantity            int       `json:"quantity"`
	TotalCost           float64   `json:"total_cost"`
	Status              string    `json:"status"` // PENDING_SELLER, CONFIRMED, READY_OR_SHIPPED, COMPLETED, CANCELLED_BY_BUYER, CANCELLED_BY_SELLER, EXPIRED
	CreatedAt           time.Time `json:"created_at"`
	DeliveryType        string    `json:"delivery_type"` // PICKUP, MY_DELIVERY, INTERCITY
	ShippingAddressText *string   `json:"shipping_address_text"`
	ShippingLat         *float64  `json:"shipping_lat"`
	ShippingLng         *float64  `json:"shipping_lng"`
	ShippingComment     *string   `json:"shipping_comment"`
	ConfirmCode         string    `json:"confirm_code"`
	// New field for intercity delivery details
	IntercityDelivery *IntercityDeliveryDetails `gorm:"serializer:json" json:"intercity_delivery,omitempty"`
}
