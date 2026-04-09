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
}
