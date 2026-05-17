package dto

import (
	"ozMadeBack/internal/models" // Import models to reuse structs
	"time"
)

// --- Intercity Delivery DTOs ---

// AddressDetails for intercity delivery estimation request
type AddressDetails struct {
	City        string  `json:"city" binding:"required"`
	FullAddress string  `json:"fullAddress" binding:"required"`
	Latitude    float64 `json:"latitude" binding:"required"`
	Longitude   float64 `json:"longitude" binding:"required"`
}

// PackageDetails for intercity delivery estimation request
type PackageDetails struct {
	WeightGrams int `json:"weightGrams" binding:"required,min=1"`
	HeightCm    int `json:"heightCm" binding:"required,min=1"`
	WidthCm     int `json:"widthCm" binding:"required,min=1"`
	DepthCm     int `json:"depthCm" binding:"required,min=1"`
}

// IntercityEstimateRequest for POST /delivery/intercity/estimate
type IntercityEstimateRequest struct {
	FromAddress AddressDetails `json:"fromAddress" binding:"required"`
	ToAddress   AddressDetails `json:"toAddress" binding:"required"`
	Package     PackageDetails `json:"package" binding:"required"`
}

// IntercityEstimateResponse for POST /delivery/intercity/estimate
type IntercityEstimateResponse struct {
	Provider          string  `json:"provider"`
	Price             float64 `json:"price"`
	Currency          string  `json:"currency"`
	MinDays           int     `json:"minDays"`
	MaxDays           int     `json:"maxDays"`
	EstimatedDateFrom string  `json:"estimatedDateFrom"`
	EstimatedDateTo   string  `json:"estimatedDateTo"`
}

// IntercityDeliveryDetailsInput for creating an order (includes receiver info)
type IntercityDeliveryDetailsInput struct {
	Provider          string         `json:"provider" binding:"required"`
	Price             float64        `json:"price" binding:"required"`
	Currency          string         `json:"currency" binding:"required"`
	MinDays           int            `json:"minDays" binding:"required"`
	MaxDays           int            `json:"maxDays" binding:"required"`
	EstimatedDateFrom string         `json:"estimatedDateFrom" binding:"required"`
	EstimatedDateTo   string         `json:"estimatedDateTo" binding:"required"`
	FromAddress       AddressDetails `json:"fromAddress" binding:"required"`
	ToAddress         AddressDetails `json:"toAddress" binding:"required"`
	Package           PackageDetails `json:"package" binding:"required"`
	ReceiverName      string         `json:"receiverName" binding:"required"`
	ReceiverPhone     string         `json:"receiverPhone" binding:"required"`
	ReceiverAddress   string         `json:"receiverAddress" binding:"required"`
}

// --- Order DTOs ---

type OrderDto struct {
	ID                  uint                             `json:"id"`
	Status              string                           `json:"status"`
	CreatedAt           time.Time                        `json:"created_at"`
	ProductID           uint                             `json:"product_id"`
	ProductTitle        string                           `json:"product_title"`
	ProductImageUrl     string                           `json:"product_image_url"`
	Price               float64                          `json:"price"`
	Quantity            int                              `json:"quantity"`
	TotalCost           float64                          `json:"total_cost"`
	SellerID            uint                             `json:"seller_id"`
	SellerName          string                           `json:"seller_name"`
	DeliveryType        string                           `json:"delivery_type"`
	PickupAddress       *string                          `json:"pickup_address"`
	PickupTime          *string                          `json:"pickup_time"`
	ZoneCenterLat       *float64                         `json:"zone_center_lat"`
	ZoneCenterLng       *float64                         `json:"zone_center_lng"`
	ZoneRadiusKm        *float64                         `json:"zone_radius_km"`
	ZoneCenterAddress   *string                          `json:"zone_center_address"`
	ShippingAddressText *string                          `json:"shipping_address_text"`
	ShippingLat         *float64                         `json:"shipping_lat"`
	ShippingLng         *float64                         `json:"shipping_lng"`
	ShippingComment     *string                          `json:"shipping_comment"`
	ConfirmCode         *string                          `json:"confirm_code"`
	IntercityDelivery   *models.IntercityDeliveryDetails `json:"intercity_delivery,omitempty"` // Include intercity details
}

type CheckoutOptionDto struct {
	Code           string   `json:"code"`
	Title          string   `json:"title"`
	Enabled        bool     `json:"enabled"`
	RequiresFields []string `json:"requires_fields"`
	Description    string   `json:"description"`
}

type CreateOrderInput struct {
	ProductID           uint                           `json:"product_id" binding:"required"`
	Quantity            int                            `json:"quantity" binding:"required,min=1"`
	DeliveryType        string                         `json:"delivery_type" binding:"required"`
	ShippingAddressText *string                        `json:"shipping_address_text"`
	ShippingLat         *float64                       `json:"shipping_lat"`
	ShippingLng         *float64                       `json:"shipping_lng"`
	ShippingComment     *string                        `json:"shipping_comment"`
	IntercityDelivery   *IntercityDeliveryDetailsInput `json:"intercity_delivery,omitempty"` // New field for intercity delivery details
}

type OrderCompleteInput struct {
	Code string `json:"code" binding:"required"`
}
