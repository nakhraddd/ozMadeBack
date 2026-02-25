package handlers

import (
	"net/http"
	"strconv"
	"time"

	"ozMadeBack/internal/database"
	"ozMadeBack/internal/models"
	"ozMadeBack/internal/services"

	"github.com/gin-gonic/gin"
)

type SellerHandler struct {
	GCSService *services.GCSService
}

func NewSellerHandler(gcsService *services.GCSService) *SellerHandler {
	return &SellerHandler{GCSService: gcsService}
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

	seller := models.Seller{
		UserID: userID,
		Status: "pending",
	}

	tx := database.DB.Begin()

	if err := tx.Create(&seller).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register seller"})
		return
	}

	if err := tx.Model(&user).Updates(map[string]interface{}{
		"is_seller": true,
		"role":      "seller",
	}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user status"})
		return
	}

	tx.Commit()

	c.JSON(http.StatusOK, gin.H{"message": "Seller application submitted"})
}

func (h *SellerHandler) GetUploadIDURL(c *gin.Context) {
	userID := c.GetUint("userID")
	objectName := "seller_ids/" + strconv.FormatUint(uint64(userID), 10) + ".jpg"
	url, err := h.GCSService.GenerateSignedURL(objectName, "PUT", 15*time.Minute)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate upload URL"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"upload_url": url})
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
	}

	if err := database.DB.Create(&product).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create product"})
		return
	}

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

	if err := database.DB.Save(&product).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update product"})
		return
	}

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

	c.JSON(http.StatusOK, gin.H{"message": "Product deleted"})
}

func (h *SellerHandler) GetProfile(c *gin.Context) {
	userID := c.GetUint("userID")
	var seller models.Seller
	if err := database.DB.Preload("User").Where("user_id = ?", userID).First(&seller).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Seller not found"})
		return
	}

	var productCount int64
	database.DB.Model(&models.Product{}).Where("seller_id = ?", seller.ID).Count(&productCount)

	c.JSON(http.StatusOK, gin.H{
		"name":           seller.User.Email,
		"status":         seller.Status,
		"total_products": productCount,
	})
}

func (h *SellerHandler) UpdateProfile(c *gin.Context) {
	userID := c.GetUint("userID")
	var seller models.Seller
	if err := database.DB.Where("user_id = ?", userID).First(&seller).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Seller not found"})
		return
	}

	var input struct {
		ProfilePicture string `json:"profile_picture"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Logic to update profile picture (e.g., generate signed URL for upload)
	// For simplicity, assuming the input contains the new profile picture URL or path
	// In a real scenario, you might generate a signed URL for the client to upload the image

	c.JSON(http.StatusOK, gin.H{"message": "Profile updated"})
}

func (h *SellerHandler) GetChats(c *gin.Context) {
	userID := c.GetUint("userID")
	var seller models.Seller
	if err := database.DB.Where("user_id = ?", userID).First(&seller).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Seller not found"})
		return
	}

	var chats []models.Chat
	if err := database.DB.Where("seller_id = ?", seller.ID).Find(&chats).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch chats"})
		return
	}

	c.JSON(http.StatusOK, chats)
}

func (h *SellerHandler) GetChatMessages(c *gin.Context) {
	chatID := c.Param("chat_id")
	var messages []models.Message
	if err := database.DB.Where("chat_id = ?", chatID).Find(&messages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch messages"})
		return
	}

	c.JSON(http.StatusOK, messages)
}

func (h *SellerHandler) GetDelivery(c *gin.Context) {
	userID := c.GetUint("userID")
	var seller models.Seller
	if err := database.DB.Where("user_id = ?", userID).First(&seller).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Seller not found"})
		return
	}

	// Set default radius if 0
	if seller.DeliveryRadiusKm == 0 {
		seller.DeliveryRadiusKm = 3
	}

	c.JSON(http.StatusOK, gin.H{
		"pickup_enabled":          seller.PickupEnabled,
		"pickup_address":          seller.PickupAddress,
		"pickup_time":             seller.PickupTime,
		"free_delivery_enabled":   seller.FreeDeliveryEnabled,
		"delivery_center_lat":     seller.DeliveryCenterLat,
		"delivery_center_lng":     seller.DeliveryCenterLng,
		"delivery_radius_km":      seller.DeliveryRadiusKm,
		"delivery_center_address": seller.DeliveryCenterAddress,
		"intercity_enabled":       seller.IntercityEnabled,
	})
}

func (h *SellerHandler) UpdateDelivery(c *gin.Context) {
	userID := c.GetUint("userID")
	var seller models.Seller
	if err := database.DB.Where("user_id = ?", userID).First(&seller).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Seller not found"})
		return
	}

	var input struct {
		PickupEnabled         bool    `json:"pickup_enabled"`
		PickupAddress         string  `json:"pickup_address"`
		PickupTime            string  `json:"pickup_time"`
		FreeDeliveryEnabled   bool    `json:"free_delivery_enabled"`
		DeliveryCenterLat     float64 `json:"delivery_center_lat"`
		DeliveryCenterLng     float64 `json:"delivery_center_lng"`
		DeliveryRadiusKm      float64 `json:"delivery_radius_km"`
		DeliveryCenterAddress string  `json:"delivery_center_address"`
		IntercityEnabled      bool    `json:"intercity_enabled"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	seller.PickupEnabled = input.PickupEnabled
	seller.PickupAddress = input.PickupAddress
	seller.PickupTime = input.PickupTime
	seller.FreeDeliveryEnabled = input.FreeDeliveryEnabled
	seller.DeliveryCenterLat = input.DeliveryCenterLat
	seller.DeliveryCenterLng = input.DeliveryCenterLng
	seller.DeliveryRadiusKm = input.DeliveryRadiusKm
	seller.DeliveryCenterAddress = input.DeliveryCenterAddress
	seller.IntercityEnabled = input.IntercityEnabled

	if err := database.DB.Save(&seller).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update delivery settings"})
		return
	}

	c.JSON(http.StatusOK, seller)
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
		if err := database.DB.Where("product_id IN ?", productIDs).Find(&orders).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch orders"})
			return
		}
	} else {
		orders = []models.Order{}
	}

	c.JSON(http.StatusOK, orders)
}

func (h *SellerHandler) ConfirmOrder(c *gin.Context) {
	orderID := c.Param("id")
	// TODO: Verify seller owns this order
	if err := database.DB.Model(&models.Order{}).Where("id = ?", orderID).Update("status", "confirmed").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to confirm order"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Order confirmed"})
}

func (h *SellerHandler) CancelOrderSeller(c *gin.Context) {
	orderID := c.Param("id")
	// TODO: Verify seller owns this order
	if err := database.DB.Model(&models.Order{}).Where("id = ?", orderID).Update("status", "cancelled").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel order"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Order cancelled"})
}

func (h *SellerHandler) ReadyOrShipped(c *gin.Context) {
	orderID := c.Param("id")
	var input struct {
		// Assuming request body might contain tracking info or just status trigger
	}
	// For now just update status
	if err := database.DB.Model(&models.Order{}).Where("id = ?", orderID).Update("status", "shipped").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update order status"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Order marked as ready/shipped"})
}

func (h *SellerHandler) CompleteOrder(c *gin.Context) {
	orderID := c.Param("id")
	if err := database.DB.Model(&models.Order{}).Where("id = ?", orderID).Update("status", "completed").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to complete order"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Order completed"})
}
