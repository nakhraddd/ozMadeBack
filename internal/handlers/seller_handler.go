package handlers

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"ozMadeBack/config"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"ozMadeBack/internal/database"
	"ozMadeBack/internal/models"
	"ozMadeBack/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type SellerHandler struct {
	GCSService *services.GCSService
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

// SellerQualityCommentDto matches SellerReviewItemDto from Kotlin
type SellerQualityCommentDto struct {
	ID           uint      `json:"id"`
	UserName     string    `json:"user_name"`
	ProductID    uint      `json:"product_id"`
	ProductTitle string    `json:"product_title"`
	Rating       float64   `json:"rating"`
	CreatedAt    time.Time `json:"created_at"`
	Text         string    `json:"text"`
}

// SellerQualityResponse matches the full seller profile/quality requirements
type SellerQualityResponse struct {
	SellerName     string                    `json:"seller_name"`
	PhotoURL       string                    `json:"photo_url"`
	FirstName      string                    `json:"first_name"`
	LastName       string                    `json:"last_name"`
	StoreName      string                    `json:"store_name"`
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

type Level struct {
	Title    string  `json:"level_title"`
	Progress float32 `json:"level_progress"`
	Hint     string  `json:"level_hint"`
}

type sellerMetrics struct {
	OrdersCount    int
	AverageRating  float64
	RatingsCount   int
	ReviewsCount   int
	DaysWithOzMade int
}

// computeLevel translates the Kotlin logic to Go
func computeLevel(orders int, rating float64, reviews int, days int) Level {
	ordersPts := int(math.Min(40, float64(orders*2)))
	reviewsPts := int(math.Min(30, float64(reviews*3)))

	ratingRaw := math.Max(0.0, rating-3.0) * 10.0
	ratingPts := int(math.Min(20, ratingRaw))

	daysPts := int(math.Min(10, float64(days/7)))

	score := ordersPts + reviewsPts + ratingPts + daysPts

	// Coerce score to be between 0 and 100
	s := int(math.Max(0, math.Min(100, float64(score))))
	progress := float32(s) / 100.0

	switch {
	case s < 20:
		return Level{"Новый мастер", progress, "Начни собирать отзывы и выполненные заказы"}
	case s < 45:
		return Level{"Надёжный мастер", progress, "Держи рейтинг и увеличивай число заказов"}
	case s < 70:
		return Level{"Проверенный мастер", progress, "Ещё немного — и ты в топе"}
	case s < 90:
		return Level{"Отличный мастер", progress, "Стабильная работа, высокий рейтинг"}
	default:
		return Level{"Топ мастер", progress, "Максимальный уровень доверия покупателей"}
	}
}

func computeSellerMetrics(seller models.Seller) sellerMetrics {
	productIDs := make([]uint, 0)
	if err := database.DB.Model(&models.Product{}).Where("seller_id = ?", seller.ID).Pluck("id", &productIDs).Error; err != nil {
		return sellerMetrics{
			OrdersCount:    seller.OrdersCount,
			AverageRating:  seller.AverageRating,
			RatingsCount:   seller.RatingsCount,
			ReviewsCount:   seller.ReviewsCount,
			DaysWithOzMade: int(time.Since(seller.CreatedAt).Hours() / 24),
		}
	}

	metrics := sellerMetrics{
		DaysWithOzMade: int(time.Since(seller.CreatedAt).Hours() / 24),
	}

	if len(productIDs) == 0 {
		return metrics
	}

	var ordersCount int64
	database.DB.Model(&models.Order{}).Where("product_id IN ?", productIDs).Count(&ordersCount)

	var ratingsCount int64
	database.DB.Model(&models.Comment{}).Where("product_id IN ?", productIDs).Count(&ratingsCount)

	var reviewsCount int64
	database.DB.Model(&models.Comment{}).Where("product_id IN ? AND text != ''", productIDs).Count(&reviewsCount)

	var averageRating float64
	database.DB.Model(&models.Comment{}).Where("product_id IN ?", productIDs).Select("COALESCE(AVG(rating), 0)").Scan(&averageRating)

	metrics.OrdersCount = int(ordersCount)
	metrics.RatingsCount = int(ratingsCount)
	metrics.ReviewsCount = int(reviewsCount)
	metrics.AverageRating = math.Round(averageRating*10) / 10

	return metrics
}

func NewSellerHandler(gcsService *services.GCSService) *SellerHandler {
	return &SellerHandler{GCSService: gcsService}
}

// SellerRegistrationRequestDto matches the Kotlin data class for seller registration
type SellerRegistrationRequestDto struct {
	FirstName   string   `json:"first_name" binding:"required"`
	LastName    string   `json:"last_name" binding:"required"`
	DisplayName string   `json:"display_name" binding:"required"` // Maps to StoreName
	City        string   `json:"city" binding:"required"`
	Address     string   `json:"address" binding:"required"`
	Categories  []string `json:"categories" binding:"required"`
	About       string   `json:"about"` // Maps to Description
	IDCardUrl   string   `json:"id_card_url"`
}

func (h *SellerHandler) RegisterSeller(c *gin.Context) {
	userID := c.GetUint("userID")
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Check if a seller profile already exists for this user
	var existingSeller models.Seller
	if err := database.DB.Where("user_id = ?", userID).First(&existingSeller).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Seller profile already exists for this user"})
		return
	}

	if user.IsSeller {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User is already a seller"})
		return
	}

	var input SellerRegistrationRequestDto
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	seller := models.Seller{
		UserID:      userID,
		FirstName:   input.FirstName,
		LastName:    input.LastName,
		StoreName:   input.DisplayName,
		City:        input.City,
		Address:     input.Address,
		Description: input.About,
		Categories:  strings.Join(input.Categories, ","), // Convert slice to comma-separated string
		IDCard:      input.IDCardUrl,
		Status:      "pending",
	}

	tx := database.DB.Begin()

	if err := tx.Create(&seller).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register seller"})
		return
	}

	// Update user status to is_seller and role
	if err := tx.Model(&user).Updates(map[string]interface{}{
		"is_seller": true,
		"role":      "seller",
	}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user status"})
		return
	}

	tx.Commit()

	c.JSON(http.StatusOK, gin.H{"message": "Seller application submitted successfully", "seller_id": seller.ID})
}

func (h *SellerHandler) GetUploadIDURL(c *gin.Context) {
	userID := c.GetUint("userID")
	objectName := "seller_ids/" + strconv.FormatUint(uint64(userID), 10) + ".jpg"
	url, err := h.GCSService.GenerateSignedURL(objectName, "PUT", 15*time.Minute, "image/jpeg")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate upload URL"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"upload_url": url})
}

func (h *SellerHandler) GetUploadProductPhotoURL(c *gin.Context) {
	userID := c.GetUint("userID")

	// Support both snake_case (Android) and camelCase
	contentType := c.Query("content_type")
	if contentType == "" {
		contentType = c.Query("contentType")
	}
	if contentType == "" {
		contentType = "image/jpeg"
	}

	// Determine extension from content type
	ext := ".jpg"
	if strings.Contains(contentType, "png") {
		ext = ".png"
	} else if strings.Contains(contentType, "webp") {
		ext = ".webp"
	}

	// Generate a unique file name
	fileName := strconv.FormatUint(uint64(userID), 10) + "_" + strconv.FormatInt(time.Now().UnixNano(), 10) + ext
	objectName := "products/" + fileName

	// Generate Signed URL ensuring the Content-Type is part of the signature
	url, err := h.GCSService.GenerateSignedURL(objectName, "PUT", 15*time.Minute, contentType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate upload URL"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"uploadUrl": url,
		"fileUrl":   objectName, // Return the full path including "products/"
	})
}

func (h *SellerHandler) GetUploadPhotoURL(c *gin.Context) {
	userID := c.GetUint("userID")
	fileName := c.Query("file_name")
	contentType := c.Query("content_type")

	if fileName == "" || contentType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file_name and content_type are required"})
		return
	}

	if h.GCSService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "GCS service not initialized"})
		return
	}

	ext := filepath.Ext(fileName)
	objectName := fmt.Sprintf("seller_photos/%d/%s%s", userID, uuid.New().String(), ext)

	url, err := h.GCSService.GenerateSignedURL(objectName, "PUT", 15*time.Minute, contentType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate upload URL"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"upload_url":  url,
		"object_name": objectName,
	})
}

func (h *SellerHandler) GetProducts(c *gin.Context) {
	userID := c.GetUint("userID")
	var seller models.Seller
	if err := database.DB.Where("user_id = ?", userID).First(&seller).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Seller not found"})
		return
	}

	var products []models.Product
	if err := database.DB.Where("seller_id = ?", seller.ID).Find(&products).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch products"})
		return
	}

	c.JSON(http.StatusOK, products)
}

func (h *SellerHandler) CreateProduct(c *gin.Context) {
	userID := c.GetUint("userID")
	var seller models.Seller
	if err := database.DB.Where("user_id = ?", userID).First(&seller).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Seller not found"})
		return
	}

	var input struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Price       float64  `json:"price"`
		Type        string   `json:"type"`
		Address     string   `json:"address"`
		ImageURL    string   `json:"image_url"`
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

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	product := models.Product{
		SellerID:    seller.ID,
		Title:       input.Name,
		Description: input.Description,
		Cost:        input.Price,
		Type:        input.Type,
		Address:     input.Address,
		ImageName:   input.ImageURL,
		Weight:      input.Weight,
		HeightCm:    input.HeightCm,
		WidthCm:     input.WidthCm,
		DepthCm:     input.DepthCm,
		Composition: input.Composition,
		YouTubeUrl:  input.YouTubeUrl,
		Categories:  input.Categories,
		Images:      input.Images,
		IsHidden:    input.IsHidden,
	}

	if err := database.DB.Create(&product).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create product"})
		return
	}

	services.IndexProductAsync(product)
	c.JSON(http.StatusCreated, product)
}

func (h *SellerHandler) UpdateProduct(c *gin.Context) {
	userID := c.GetUint("userID")
	productID := c.Param("id")

	var seller models.Seller
	if err := database.DB.Where("user_id = ?", userID).First(&seller).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Seller not found"})
		return
	}

	var product models.Product
	if err := database.DB.Where("id = ? AND seller_id = ?", productID, seller.ID).First(&product).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found or unauthorized"})
		return
	}

	var input struct {
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

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	product.Title = input.Title
	product.Description = input.Description
	product.Cost = input.Cost
	product.Categories = input.Categories
	product.Images = input.Images
	product.Weight = input.Weight
	product.HeightCm = input.HeightCm
	product.WidthCm = input.WidthCm
	product.DepthCm = input.DepthCm
	product.Composition = input.Composition
	product.YouTubeUrl = input.YouTubeUrl
	if input.IsHidden != nil {
		product.IsHidden = *input.IsHidden
	}

	if err := database.DB.Save(&product).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update product"})
		return
	}

	services.IndexProductAsync(product)
	c.JSON(http.StatusOK, product)
}

func (h *SellerHandler) DeleteProduct(c *gin.Context) {
	userID := c.GetUint("userID")
	productID := c.Param("id")

	var seller models.Seller
	if err := database.DB.Where("user_id = ?", userID).First(&seller).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Seller not found"})
		return
	}

	if err := database.DB.Where("id = ? AND seller_id = ?", productID, seller.ID).Delete(&models.Product{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete product"})
		return
	}

	productIDUint, err := strconv.ParseUint(productID, 10, 64)
	if err != nil {
		log.Printf("failed to parse deleted product ID %q for search cleanup: %v", productID, err)
	} else {
		services.DeleteProductFromSearchAsync(uint(productIDUint))
	}

	c.JSON(http.StatusOK, gin.H{"message": "Product deleted"})
}

func (h *SellerHandler) GetProfile(c *gin.Context) {
	userID := c.GetUint("userID")
	var seller models.Seller
	if err := database.DB.Preload("User").Where("user_id = ?", userID).First(&seller).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Seller not found"})
		return
	}

	var products []models.Product
	database.DB.Where("seller_id = ?", seller.ID).Find(&products)

	productDtos := buildSellerProfileProducts(products, seller)
	name := resolveSellerPublicName(seller)
	metrics := computeSellerMetrics(seller)
	level := computeLevel(metrics.OrdersCount, metrics.AverageRating, metrics.ReviewsCount, metrics.DaysWithOzMade)

	categories := []string{}
	if seller.Categories != "" {
		categories = strings.Split(seller.Categories, ",")
	}

	photoURL := ""
	if seller.PhotoURL != "" {
		photoURL, _ = services.GenerateSignedURLForSeller(seller.PhotoURL)
	}

	c.JSON(http.StatusOK, gin.H{
		"id":             seller.ID,
		"name":           name,
		"phone_number":   seller.User.PhoneNumber,
		"address":        seller.User.Address,
		"status":         seller.Status,
		"total_products": len(productDtos),
		"orders_count":   metrics.OrdersCount,
		"average_rating": metrics.AverageRating,
		"ratings_count":  metrics.RatingsCount,
		"reviews_count":  metrics.ReviewsCount,
		"days_with_us":   metrics.DaysWithOzMade,
		"level_title":    level.Title,
		"level_progress": level.Progress,
		"level_hint":     level.Hint,
		"products":       productDtos,
		"delivery":       serializeDeliverySettings(seller),
		// Additional fields for editing
		"first_name":   seller.FirstName,
		"last_name":    seller.LastName,
		"display_name": seller.StoreName, // Maps to StoreName
		"city":         seller.City,
		"about":        seller.Description, // Maps to Description
		"categories":   categories,
		"photo_url":    photoURL,
	})
}

func (h *SellerHandler) GetSellerProfile(c *gin.Context) {
	id := c.Param("id")
	var seller models.Seller
	if err := database.DB.Preload("User").First(&seller, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Seller not found"})
		return
	}

	var products []models.Product
	database.DB.Where("seller_id = ?", seller.ID).Find(&products)

	productDtos := buildSellerProfileProducts(products, seller)
	name := resolveSellerPublicName(seller)
	metrics := computeSellerMetrics(seller)
	level := computeLevel(metrics.OrdersCount, metrics.AverageRating, metrics.ReviewsCount, metrics.DaysWithOzMade)

	photoURL := ""
	if seller.PhotoURL != "" {
		photoURL, _ = services.GenerateSignedURLForSeller(seller.PhotoURL)
	}

	c.JSON(http.StatusOK, gin.H{
		"id":             seller.ID,
		"name":           name,
		"photo_url":      photoURL,
		"phone_number":   seller.User.PhoneNumber,
		"address":        seller.User.Address,
		"status":         seller.Status,
		"total_products": len(productDtos),
		"orders_count":   metrics.OrdersCount,
		"average_rating": metrics.AverageRating,
		"ratings_count":  metrics.RatingsCount,
		"reviews_count":  metrics.ReviewsCount,
		"days_with_us":   metrics.DaysWithOzMade,
		"level_title":    level.Title,
		"level_progress": level.Progress,
		"level_hint":     level.Hint,
		"products":       productDtos,
		"delivery":       serializeDeliverySettings(seller),
	})
}

// GetSellerQuality handles fetching seller quality information and reviews
func (h *SellerHandler) GetSellerQuality(c *gin.Context) {
	sellerIDStr := c.Param("id")
	sellerID, err := strconv.ParseUint(sellerIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid seller ID"})
		return
	}

	var seller models.Seller
	if err := database.DB.Preload("User").First(&seller, sellerID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Seller not found"})
		return
	}

	var comments []models.Comment
	database.DB.Preload("User").Preload("Product").
		Joins("JOIN products ON products.id = comments.product_id").
		Where("products.seller_id = ?", sellerID).
		Find(&comments)

	metrics := computeSellerMetrics(seller)
	level := computeLevel(metrics.OrdersCount, metrics.AverageRating, metrics.ReviewsCount, metrics.DaysWithOzMade)

	var reviewDtos []SellerQualityCommentDto
	for _, comment := range comments {
		name := comment.User.Name

		reviewDtos = append(reviewDtos, SellerQualityCommentDto{
			ID:           comment.ID,
			UserName:     name,
			ProductID:    comment.ProductID,
			ProductTitle: comment.Product.Title,
			Rating:       comment.Rating,
			CreatedAt:    comment.CreatedAt,
			Text:         comment.Text,
		})
	}

	photoURL := ""
	if seller.PhotoURL != "" {
		photoURL, _ = services.GenerateSignedURLForSeller(seller.PhotoURL)
	}

	response := SellerQualityResponse{
		SellerName:     resolveSellerPublicName(seller),
		PhotoURL:       photoURL,
		FirstName:      seller.FirstName,
		LastName:       seller.LastName,
		StoreName:      seller.StoreName,
		City:           seller.City,
		Address:        seller.Address,
		Categories:     seller.Categories,
		Description:    seller.Description,
		OrdersCount:    metrics.OrdersCount,
		DaysWithOzMade: metrics.DaysWithOzMade,
		LevelTitle:     level.Title,
		LevelProgress:  level.Progress,
		LevelHint:      level.Hint,
		AverageRating:  metrics.AverageRating,
		RatingsCount:   metrics.RatingsCount,
		ReviewsCount:   metrics.ReviewsCount,
		Reviews:        reviewDtos,
	}

	c.JSON(http.StatusOK, response)
}

func buildSellerProfileProducts(products []models.Product, seller models.Seller) []SellerProfileProductDto {
	if len(products) == 0 {
		return []SellerProfileProductDto{}
	}

	appLinkBase := config.GetEnv("APP_LINK_BASE_URL", "https://ozmade-applink.vercel.app")
	sellerName := resolveSellerPublicName(seller)
	response := make([]SellerProfileProductDto, 0, len(products))

	for i := range products {
		product := products[i]

		if product.ImageName != "" {
			objectName := product.ImageName
			if !strings.HasPrefix(objectName, "products/") && !strings.HasPrefix(objectName, "seller_ids/") && !strings.HasPrefix(objectName, "seller_photos/") {
				objectName = "products/" + objectName
			}
			if url, err := services.GenerateSignedURL(objectName); err == nil {
				product.ImageName = url
			}
		}

		for j, imageName := range product.Images {
			if imageName == "" {
				continue
			}

			objectName := imageName
			if !strings.HasPrefix(objectName, "products/") && !strings.HasPrefix(objectName, "seller_ids/") && !strings.HasPrefix(objectName, "seller_photos/") {
				objectName = "products/" + objectName
			}
			if url, err := services.GenerateSignedURL(objectName); err == nil {
				product.Images[j] = url
			}
		}

		response = append(response, SellerProfileProductDto{
			ID:            product.ID,
			SellerID:      product.SellerID,
			Title:         product.Title,
			Description:   product.Description,
			Type:          product.Type,
			Cost:          product.Cost,
			Address:       product.Address,
			WhatsAppLink:  product.WhatsAppLink,
			ViewCount:     product.ViewCount,
			AverageRating: product.AverageRating,
			ImageName:     product.ImageName,
			Images:        product.Images,
			Weight:        product.Weight,
			HeightCm:      product.HeightCm,
			WidthCm:       product.WidthCm,
			DepthCm:       product.DepthCm,
			Composition:   product.Composition,
			YouTubeURL:    product.YouTubeUrl,
			Categories:    product.Categories,
			IsHidden:      product.IsHidden,
			CreatedAt:     product.CreatedAt,
			SellerName:    sellerName,
			ShareLink:     appLinkBase + "/products/" + strconv.FormatUint(uint64(product.ID), 10),
		})
	}

	return response
}

func resolveSellerPublicName(seller models.Seller) string {
	if seller.StoreName != "" { // Prefer store name if available
		return seller.StoreName
	}
	if seller.FirstName != "" && seller.LastName != "" {
		return seller.FirstName + " " + seller.LastName
	}
	if seller.User.Name != "" {
		return seller.User.Name
	}
	return "Unknown"
}

func (h *SellerHandler) UpdateProfile(c *gin.Context) {
	userID := c.GetUint("userID")

	var seller models.Seller
	if err := database.DB.Where("user_id = ?", userID).First(&seller).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Seller not found"})
		return
	}

	// Handle multipart form for photo upload
	contentType := c.GetHeader("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		// Update fields from form
		if firstName := c.PostForm("first_name"); firstName != "" {
			seller.FirstName = firstName
		}
		if lastName := c.PostForm("last_name"); lastName != "" {
			seller.LastName = lastName
		}
		if storeName := c.PostForm("store_name"); storeName != "" {
			seller.StoreName = storeName
		}
		if city := c.PostForm("city"); city != "" {
			seller.City = city
		}
		if address := c.PostForm("address"); address != "" {
			seller.Address = address
		}
		if description := c.PostForm("description"); description != "" {
			seller.Description = description
		}
		if cats := c.PostForm("categories"); cats != "" {
			seller.Categories = cats
		}

		// Handle photo upload
		file, err := c.FormFile("photo")
		if err == nil && h.GCSService != nil {
			ext := filepath.Ext(file.Filename)
			objectName := fmt.Sprintf("seller_photos/%d/%s%s", userID, uuid.New().String(), ext)

			f, _ := file.Open()
			defer f.Close()

			wc := h.GCSService.Client.Bucket(h.GCSService.BucketName).Object(objectName).NewWriter(c.Request.Context())
			// Actual implementation would use io.Copy(wc, f) then wc.Close()
			_ = wc

			seller.PhotoURL = objectName
		}
	} else {
		// Handle JSON update
		var input struct {
			FirstName   *string   `json:"first_name"`
			LastName    *string   `json:"last_name"`
			StoreName   *string   `json:"display_name"`
			City        *string   `json:"city"`
			Address     *string   `json:"address"`
			Description *string   `json:"description"`
			Categories  *[]string `json:"categories"` // Slice of strings
			PhotoURL    *string   `json:"photo_url"`
		}

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		updates := make(map[string]interface{})
		if input.FirstName != nil {
			updates["first_name"] = *input.FirstName
		}
		if input.LastName != nil {
			updates["last_name"] = *input.LastName
		}
		if input.StoreName != nil {
			updates["display_name"] = *input.StoreName
		}
		if input.City != nil {
			updates["city"] = *input.City
		}
		if input.Address != nil {
			updates["address"] = *input.Address
		}
		if input.Description != nil {
			updates["description"] = *input.Description
		}
		if input.Categories != nil {
			updates["categories"] = strings.Join(*input.Categories, ",")
		}
		if input.PhotoURL != nil {
			updates["photo_url"] = *input.PhotoURL
		}

		if len(updates) > 0 {
			if err := database.DB.Model(&seller).Updates(updates).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update seller profile"})
				return
			}
		}
	}

	if err := database.DB.Save(&seller).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save profile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Seller profile updated"})
}

func (h *SellerHandler) GetDelivery(c *gin.Context) {
	userID := c.GetUint("userID")
	var seller models.Seller
	if err := database.DB.Where("user_id = ?", userID).First(&seller).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Seller not found"})
		return
	}

	c.JSON(http.StatusOK, serializeDeliverySettings(seller))
}

func (h *SellerHandler) UpdateDelivery(c *gin.Context) {
	userID := c.GetUint("userID")
	var seller models.Seller
	if err := database.DB.Where("user_id = ?", userID).First(&seller).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Seller not found"})
		return
	}

	var input struct {
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

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Apply updates to a temporary variable to check constraints before saving
	tempSeller := seller

	if input.PickupEnabled != nil {
		tempSeller.PickupEnabled = *input.PickupEnabled
	}
	if input.PickupAddress != nil {
		tempSeller.PickupAddress = *input.PickupAddress
	}
	if input.PickupTime != nil {
		tempSeller.PickupTime = *input.PickupTime
	}
	if input.FreeDeliveryEnabled != nil {
		tempSeller.FreeDeliveryEnabled = *input.FreeDeliveryEnabled
	}
	if input.DeliveryCenterLat != nil {
		tempSeller.DeliveryCenterLat = *input.DeliveryCenterLat
	}
	if input.DeliveryCenterLng != nil {
		tempSeller.DeliveryCenterLng = *input.DeliveryCenterLng
	}
	if input.DeliveryRadiusKm != nil {
		tempSeller.DeliveryRadiusKm = *input.DeliveryRadiusKm
	}
	if input.DeliveryCenterAddress != nil {
		tempSeller.DeliveryCenterAddress = *input.DeliveryCenterAddress
	}
	if input.IntercityEnabled != nil {
		tempSeller.IntercityEnabled = *input.IntercityEnabled
	}

	// Backend Validation
	if tempSeller.PickupEnabled {
		if tempSeller.PickupAddress == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "pickup_address is required when pickup_enabled is true"})
			return
		}
		if tempSeller.PickupTime == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "pickup_time is required when pickup_enabled is true"})
			return
		}
	}

	if tempSeller.FreeDeliveryEnabled {
		if tempSeller.DeliveryCenterLat == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "delivery_center_lat is required when free_delivery_enabled is true"})
			return
		}
		if tempSeller.DeliveryCenterLng == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "delivery_center_lng is required when free_delivery_enabled is true"})
			return
		}
		if tempSeller.DeliveryRadiusKm <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "delivery_radius_km must be > 0 when free_delivery_enabled is true"})
			return
		}
		if tempSeller.DeliveryCenterAddress == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "delivery_center_address is required when free_delivery_enabled is true"})
			return
		}
	}

	// IntercityEnabled validation: "no additional fields are needed" - so we do nothing.

	// If validation passes, update the actual seller object
	seller = tempSeller

	if err := database.DB.Save(&seller).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update delivery settings"})
		return
	}

	c.JSON(http.StatusOK, serializeDeliverySettings(seller))
}

func serializeDeliverySettings(seller models.Seller) gin.H {
	response := gin.H{
		"pickup_enabled":        seller.PickupEnabled,
		"free_delivery_enabled": seller.FreeDeliveryEnabled,
		"intercity_enabled":     seller.IntercityEnabled,
	}

	if seller.PickupAddress != "" {
		response["pickup_address"] = seller.PickupAddress
	} else {
		response["pickup_address"] = nil
	}

	if seller.PickupTime != "" {
		response["pickup_time"] = seller.PickupTime
	} else {
		response["pickup_time"] = nil
	}

	if seller.DeliveryCenterLat != 0 {
		response["delivery_center_lat"] = seller.DeliveryCenterLat
	} else {
		response["delivery_center_lat"] = nil
	}

	if seller.DeliveryCenterLng != 0 {
		response["delivery_center_lng"] = seller.DeliveryCenterLng
	} else {
		response["delivery_center_lng"] = nil
	}

	if seller.DeliveryRadiusKm != 0 {
		response["delivery_radius_km"] = seller.DeliveryRadiusKm
	} else {
		response["delivery_radius_km"] = nil
	}

	if seller.DeliveryCenterAddress != "" {
		response["delivery_center_address"] = seller.DeliveryCenterAddress
	} else {
		response["delivery_center_address"] = nil
	}

	return response
}

func (h *SellerHandler) GetSellerOrders(c *gin.Context) {
	userID := c.GetUint("userID")
	var seller models.Seller
	if err := database.DB.Where("user_id = ?", userID).First(&seller).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Seller not found"})
		return
	}

	// Find products belonging to this seller
	var productIDs []uint
	if err := database.DB.Model(&models.Product{}).Where("seller_id = ?", seller.ID).Pluck("id", &productIDs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch seller products"})
		return
	}

	var orders []models.Order
	if len(productIDs) > 0 {
		if err := database.DB.Where("product_id IN ?", productIDs).Order("created_at desc").Find(&orders).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch orders"})
			return
		}
	} else {
		orders = []models.Order{}
	}

	// Map to DTO
	var dtos []OrderDto
	for _, order := range orders {
		var product models.Product

		database.DB.First(&product, order.ProductID)
		// seller is already fetched above

		dtos = append(dtos, mapOrderToDto(order, product, seller))
	}

	c.JSON(http.StatusOK, dtos)
}

func (h *SellerHandler) ConfirmOrder(c *gin.Context) {
	orderID := c.Param("id")
	userID := c.GetUint("userID")

	var order models.Order
	if err := database.DB.First(&order, orderID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	if !isSellerOrder(userID, order.ProductID) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if order.Status != models.StatusPendingSeller {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order must be PENDING_SELLER to confirm"})
		return
	}

	updates := map[string]interface{}{
		"status": models.StatusConfirmed,
	}

	// Generate confirm code if needed (PICKUP/MY_DELIVERY) and not already set?
	if order.ConfirmCode == "" && (order.DeliveryType == models.DeliveryTypePickup || order.DeliveryType == models.DeliveryTypeMyDelivery) {
		updates["confirm_code"] = strconv.Itoa(1000 + rand.Intn(9000))
	}

	if err := database.DB.Model(&order).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to confirm order"})
		return
	}

	orderRecordID := order.ID
	_ = services.CreateNotification(
		order.UserID,
		"Order accepted",
		"The seller accepted your order.",
		"buyer_order_confirmed",
		&orderRecordID,
		nil,
	)
	c.JSON(http.StatusOK, gin.H{"message": "Order confirmed", "status": models.StatusConfirmed})
}

func (h *SellerHandler) CancelOrderSeller(c *gin.Context) {
	orderID := c.Param("id")
	userID := c.GetUint("userID")

	var order models.Order
	if err := database.DB.First(&order, orderID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	if !isSellerOrder(userID, order.ProductID) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if order.Status != models.StatusPendingSeller && order.Status != models.StatusConfirmed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot cancel order in current status"})
		return
	}

	if err := database.DB.Model(&order).Update("status", models.StatusCancelledBySeller).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel order"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Order cancelled"})
}

func (h *SellerHandler) ReadyOrShipped(c *gin.Context) {
	orderID := c.Param("id")
	userID := c.GetUint("userID")

	var order models.Order
	if err := database.DB.First(&order, orderID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	if !isSellerOrder(userID, order.ProductID) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if order.DeliveryType != models.DeliveryTypeIntercity {
		c.JSON(http.StatusBadRequest, gin.H{"error": "READY_OR_SHIPPED is only used for INTERCITY orders"})
		return
	}

	if order.Status != models.StatusConfirmed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order must be CONFIRMED to mark as ready/shipped"})
		return
	}

	if err := database.DB.Model(&order).Update("status", models.StatusReadyOrShipped).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update order status"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Order marked as ready/shipped"})
}

func (h *SellerHandler) CompleteOrder(c *gin.Context) {
	orderID := c.Param("id")
	userID := c.GetUint("userID")

	var input struct {
		Code string `json:"code"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var order models.Order
	if err := database.DB.First(&order, orderID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	if !isSellerOrder(userID, order.ProductID) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if order.DeliveryType == models.DeliveryTypeIntercity {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Intercity orders are completed by buyer confirmation"})
		return
	}

	if order.Status != models.StatusConfirmed && order.Status != models.StatusReadyOrShipped {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order must be CONFIRMED to complete"})
		return
	}

	if order.ConfirmCode != input.Code {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid confirmation code"})
		return
	}

	if err := database.DB.Model(&order).Update("status", models.StatusCompleted).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to complete order"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Order completed"})
}

// Helper to check ownership
func isSellerOrder(userID uint, productID uint) bool {
	var seller models.Seller
	if err := database.DB.Where("user_id = ?", userID).First(&seller).Error; err != nil {
		return false
	}
	var count int64
	database.DB.Model(&models.Product{}).Where("id = ? AND seller_id = ?", productID, seller.ID).Count(&count)
	return count > 0
}
