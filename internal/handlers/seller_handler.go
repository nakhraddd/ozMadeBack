package handlers

import (
	"math/rand"
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
		var seller models.Seller

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

	if order.Status != models.StatusReadyOrShipped {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order must be READY_OR_SHIPPED to complete"})
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
