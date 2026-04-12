package dto

import (
	"ozMadeBack/internal/models"
	"time"

	"github.com/gin-gonic/gin"
)

type ProductResponse struct {
	models.Product
	Delivery  gin.H  `json:"delivery"`
	Seller    gin.H  `json:"seller"`
	ShareLink string `json:"share_link"`
}

type CreateProductInput struct {
	Title       string   `json:"title" binding:"required"`
	Description string   `json:"description"`
	Price       float64  `json:"price" binding:"required"`
	Type        string   `json:"type" binding:"required"`
	Address     string   `json:"address"`
	ImageURL    string   `json:"image_url" binding:"required"`
	Weight      string   `json:"weight"`
	HeightCm    string   `json:"height_cm"`
	WidthCm     string   `json:"width_cm"`
	DepthCm     string   `json:"depth_cm"`
	Composition string   `json:"composition"`
	YouTubeUrl  string   `json:"youtube_url"`
	Categories  []string `json:"categories"`
	Images      []string `json:"images"`
	IsHidden    bool     `json:"is_hidden"`
}

type UpdateProductInput struct {
	Title       string   `json:"Title"`
	Description string   `json:"Description"`
	Cost        float64  `json:"Cost"`
	Categories  []string `json:"Categories"`
	Images      []string `json:"Images"`
	Weight      string   `json:"Weight"`
	HeightCm    string   `json:"HeightCm"`
	WidthCm     string   `json:"WidthCm"`
	DepthCm     string   `json:"DepthCm"`
	Composition string   `json:"Composition"`
	YouTubeUrl  string   `json:"YouTubeUrl"`
	IsHidden    *bool    `json:"IsHidden"`
}

type SellerProfileProductDto struct {
	ID            uint      `json:"id"`
	SellerID      uint      `json:"seller_id"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	Type          string    `json:"type"`
	Cost          float64   `json:"cost"`
	Address       string    `json:"address"`
	WhatsAppLink  string    `json:"whats_app_link"`
	ViewCount     int64     `json:"view_count"`
	AverageRating float64   `json:"average_rating"`
	ImageName     string    `json:"image_name"`
	Images        []string  `json:"images"`
	Weight        string    `json:"weight"`
	HeightCm      string    `json:"height_cm"`
	WidthCm       string    `json:"width_cm"`
	DepthCm       string    `json:"depth_cm"`
	Composition   string    `json:"composition"`
	YouTubeURL    string    `json:"you_tube_url"`
	Categories    []string  `json:"categories"`
	IsHidden      bool      `json:"is_hidden"`
	CreatedAt     time.Time `json:"created_at"`
	SellerName    string    `json:"seller_name"`
	ShareLink     string    `json:"share_link"`
}

type ProductCheckoutOptionsResponse struct {
	ProductID            uint                `json:"product_id"`
	ProductTitle         string              `json:"product_title"`
	ProductPrice         float64             `json:"product_price"`
	ProductImageUrl      string              `json:"product_image_url"`
	SellerID             uint                `json:"seller_id"`
	SellerName           string              `json:"seller_name"`
	BuyerSavedAddress    string              `json:"buyer_saved_address"`
	BuyerSavedAddressLat *float64            `json:"buyer_saved_address_lat"`
	BuyerSavedAddressLng *float64            `json:"buyer_saved_address_lng"`
	DeliveryOptions      []CheckoutOptionDto `json:"delivery_options"`
	DeliverySummary      gin.H               `json:"delivery_summary"`
}
