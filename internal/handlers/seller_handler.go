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
		Name        string  `json:"name"`
		Description string  `json:"description"`
		Price       float64 `json:"price"`
		Type        string  `json:"type"`
		Address     string  `json:"address"`
		ImageURL    string  `json:"image_url"`
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
		Name        string  `json:"name"`
		Description string  `json:"description"`
		Price       float64 `json:"price"`
		Type        string  `json:"type"`
		Address     string  `json:"address"`
		ImageURL    string  `json:"image_url"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	product.Title = input.Name
	product.Description = input.Description
	product.Cost = input.Price
	product.Type = input.Type
	product.Address = input.Address
	product.ImageName = input.ImageURL

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
