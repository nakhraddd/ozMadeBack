package handlers

import (
	"net/http"
	"ozMadeBack/internal/database"
	"ozMadeBack/internal/models"
	"ozMadeBack/internal/services"
	"time"

	"github.com/gin-gonic/gin"
)

func SendMessage(c *gin.Context) {
	userID := c.GetUint("userID")
	chatID := c.Param("chat_id")

	var input struct {
		Content string `json:"content"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var chat models.Chat
	if err := database.DB.First(&chat, chatID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chat not found"})
		return
	}

	// Determine sender role
	senderRole := ""

	// Check if user is the buyer
	if chat.BuyerID == userID {
		senderRole = "BUYER"
	} else {
		// Check if user is the seller
		var seller models.Seller
		if err := database.DB.Where("id = ?", chat.SellerID).First(&seller).Error; err == nil {
			if seller.UserID == userID {
				senderRole = "SELLER"
			}
		}
	}

	if senderRole == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized"})
		return
	}

	message := models.Message{
		ChatID:     chat.ID,
		SenderID:   userID,
		SenderRole: senderRole,
		Content:    input.Content,
	}

	if err := database.DB.Create(&message).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send message"})
		return
	}

	c.JSON(http.StatusCreated, message)
}

func InitiateChat(c *gin.Context) {
	userID := c.GetUint("userID")
	var input struct {
		ProductID uint   `json:"product_id"`
		Content   string `json:"content"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 1. Get Product to find Seller
	var product models.Product
	if err := database.DB.First(&product, input.ProductID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	// 2. Find Seller
	var seller models.Seller
	if err := database.DB.Where("id = ?", product.SellerID).First(&seller).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Seller not found"})
		return
	}

	// Prevent chatting with yourself
	if seller.UserID == userID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot chat with yourself"})
		return
	}

	// 3. Check if chat exists
	var chat models.Chat
	err := database.DB.Where("buyer_id = ? AND seller_id = ? AND product_id = ?", userID, seller.ID, product.ID).First(&chat).Error

	if err != nil {
		// Create new chat
		chat = models.Chat{
			BuyerID:   userID,
			SellerID:  seller.ID,
			ProductID: product.ID,
		}
		if err := database.DB.Create(&chat).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create chat"})
			return
		}
	}

	// 4. Create Message if content provided
	if input.Content != "" {
		message := models.Message{
			ChatID:     chat.ID,
			SenderID:   userID,
			SenderRole: "BUYER",
			Content:    input.Content,
		}
		database.DB.Create(&message)
	}

	// Populate transient fields for response
	chat.ProductName = product.Title
	// Assuming GCS service is available via a global or we need to inject it.
	// Since this is a function, we can't easily inject without changing signature or using global.
	// internal/services/storage.go has GenerateSignedURL global helper.
	url, _ := services.GenerateSignedURL(product.ImageName)
	chat.ProductImage = url

	c.JSON(http.StatusOK, chat)
}

func GetChats(c *gin.Context) {
	userID := c.GetUint("userID")
	var chats []models.Chat
	if err := database.DB.Where("buyer_id = ?", userID).Find(&chats).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch chats"})
		return
	}

	for i := range chats {
		if chats[i].ProductID != 0 {
			var product models.Product
			if err := database.DB.First(&product, chats[i].ProductID).Error; err == nil {
				chats[i].ProductName = product.Title
				url, _ := services.GenerateSignedURL(product.ImageName)
				chats[i].ProductImage = url
			}
		}
	}

	c.JSON(http.StatusOK, chats)
}

func GetChatMessages(c *gin.Context) {
	userID := c.GetUint("userID")
	chatID := c.Param("chat_id")

	var chat models.Chat
	if err := database.DB.First(&chat, chatID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chat not found"})
		return
	}

	// Check participation
	isParticipant := false
	if chat.BuyerID == userID {
		isParticipant = true
	} else {
		var seller models.Seller
		if err := database.DB.Where("id = ?", chat.SellerID).First(&seller).Error; err == nil {
			if seller.UserID == userID {
				isParticipant = true
			}
		}
	}

	if !isParticipant {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized"})
		return
	}

	var messages []models.Message
	if err := database.DB.Where("chat_id = ?", chatID).Find(&messages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch messages"})
		return
	}

	c.JSON(http.StatusOK, messages)
}
