package dto

import "time"

type SellerRegistrationInput struct {
	FirstName   string   `json:"first_name" binding:"required"`
	LastName    string   `json:"last_name" binding:"required"`
	DisplayName string   `json:"display_name" binding:"required"`
	City        string   `json:"city" binding:"required"`
	Address     string   `json:"address" binding:"required"`
	Categories  []string `json:"categories" binding:"required"`
	About       string   `json:"about"`
	IDCardUrl   string   `json:"id_card_url"`
}

type SellerUpdateInput struct {
	FirstName   *string   `json:"first_name"`
	LastName    *string   `json:"last_name"`
	DisplayName *string   `json:"display_name"`
	City        *string   `json:"city"`
	Address     *string   `json:"address"`
	Description *string   `json:"description"`
	Categories  *[]string `json:"categories"`
	PhotoURL    *string   `json:"photo_url"`
}

type SellerQualityCommentDto struct {
	ID           uint      `json:"id"`
	UserName     string    `json:"user_name"`
	ProductID    uint      `json:"product_id"`
	ProductTitle string    `json:"product_title"`
	Rating       float64   `json:"rating"`
	CreatedAt    time.Time `json:"created_at"`
	Text         string    `json:"text"`
}

type SellerQualityResponse struct {
	SellerName     string                    `json:"seller_name"`
	PhotoURL       string                    `json:"photo_url"`
	FirstName      string                    `json:"first_name"`
	LastName       string                    `json:"last_name"`
	DisplayName    string                    `json:"display_name"`
	City           string                    `json:"city"`
	Address        string                    `json:"address"`
	Categories     string                    `json:"categories"`
	Description    string                    `json:"description"`
	OrdersCount    int                       `json:"orders_count"`
	DaysWithOzMade int                       `json:"days_with_ozmade"`
	LevelTitle     string                    `json:"level_title"`
	LevelProgress  float32                   `json:"level_progress"`
	LevelHint      string                    `json:"level_hint"`
	AverageRating  float64                   `json:"average_rating"`
	RatingsCount   int                       `json:"ratings_count"`
	ReviewsCount   int                       `json:"reviews_count"`
	Reviews        []SellerQualityCommentDto `json:"reviews"`
}

type SellerDeliveryUpdateInput struct {
	PickupEnabled         *bool    `json:"pickup_enabled"`
	PickupAddress         *string  `json:"pickup_address"`
	PickupTime            *string  `json:"pickup_time"`
	FreeDeliveryEnabled   *bool    `json:"free_delivery_enabled"`
	DeliveryCenterLat     *float64 `json:"delivery_center_lat"`
	DeliveryCenterLng     *float64 `json:"delivery_center_lng"`
	DeliveryRadiusKm      *float64 `json:"delivery_radius_km"`
	DeliveryCenterAddress *string  `json:"delivery_center_address"`
	IntercityEnabled      *bool    `json:"intercity_enabled"`
}
