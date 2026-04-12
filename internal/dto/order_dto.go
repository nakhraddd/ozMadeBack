package dto

import "time"

type OrderDto struct {
	ID                  uint      `json:"id"`
	Status              string    `json:"status"`
	CreatedAt           time.Time `json:"created_at"`
	ProductID           uint      `json:"product_id"`
	ProductTitle        string    `json:"product_title"`
	ProductImageUrl     string    `json:"product_image_url"`
	Price               float64   `json:"price"`
	Quantity            int       `json:"quantity"`
	TotalCost           float64   `json:"total_cost"`
	SellerID            uint      `json:"seller_id"`
	SellerName          string    `json:"seller_name"`
	DeliveryType        string    `json:"delivery_type"`
	PickupAddress       *string   `json:"pickup_address"`
	PickupTime          *string   `json:"pickup_time"`
	ZoneCenterLat       *float64  `json:"zone_center_lat"`
	ZoneCenterLng       *float64  `json:"zone_center_lng"`
	ZoneRadiusKm        *float64  `json:"zone_radius_km"`
	ZoneCenterAddress   *string   `json:"zone_center_address"`
	ShippingAddressText *string   `json:"shipping_address_text"`
	ShippingLat         *float64  `json:"shipping_lat"`
	ShippingLng         *float64  `json:"shipping_lng"`
	ShippingComment     *string   `json:"shipping_comment"`
	ConfirmCode         *string   `json:"confirm_code"`
}

type CheckoutOptionDto struct {
	Code           string   `json:"code"`
	Title          string   `json:"title"`
	Enabled        bool     `json:"enabled"`
	RequiresFields []string `json:"requires_fields"`
	Description    string   `json:"description"`
}

type CreateOrderInput struct {
	ProductID           uint     `json:"product_id" binding:"required"`
	Quantity            int      `json:"quantity" binding:"required,min=1"`
	DeliveryType        string   `json:"delivery_type" binding:"required"`
	ShippingAddressText *string  `json:"shipping_address_text"`
	ShippingLat         *float64 `json:"shipping_lat"`
	ShippingLng         *float64 `json:"shipping_lng"`
	ShippingComment     *string  `json:"shipping_comment"`
}

type OrderCompleteInput struct {
	Code string `json:"code" binding:"required"`
}
