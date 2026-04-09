package handlers

import (
	"encoding/json"
	"net/http"
	"ozMadeBack/internal/database"
	"ozMadeBack/internal/models"
	"ozMadeBack/internal/services"
	"ozMadeBack/pkg/realtime"
	"strconv"

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
	var recipientID uint

	// Check if user is the buyer
	if chat.BuyerID == userID {
		senderRole = "BUYER"
		var seller models.Seller
		if err := database.DB.First(&seller, chat.SellerID).Error; err == nil {
			recipientID = seller.UserID
		}
	} else {
		// Check if user is the seller
		var seller models.Seller
		if err := database.DB.Where("id = ?", chat.SellerID).First(&seller).Error; err == nil {
			if seller.UserID == userID {
				senderRole = "SELLER"
				recipientID = chat.BuyerID
			}
		}
	}

	if senderRole == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized"})
		return
	}

	// Reset deleted flags if a new message is sent
	updates := make(map[string]interface{})
	if chat.DeletedByBuyer {
		updates["deleted_by_buyer"] = false
	}
	if chat.DeletedBySeller {
		updates["deleted_by_seller"] = false
	}
	if len(updates) > 0 {
		database.DB.Model(&chat).Updates(updates)
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

	// Send real-time notification via WebSocket
	hub := realtime.GetHub()
	notification, _ := json.Marshal(gin.H{
		"type":    "new_message",
		"chat_id": chat.ID,
		"message": message,
	})
	hub.SendToUser(recipientID, notification)

	// Send FCM push notification
	var recipient models.User
	if err := database.DB.First(&recipient, recipientID).Error; err == nil && recipient.FCMToken != "" {
		_ = realtime.SendFCMNotification(
			recipient.FCMToken,
			"New Message",
			input.Content, // Or "You have a new message" for privacy
			map[string]string{
				"chat_id": strconv.Itoa(int(chat.ID)),
				"type":    "chat_message",
			},
		)
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
	} else {
		// If chat was previously deleted by the buyer (who is initiating), reset it
		if chat.DeletedByBuyer {
			database.DB.Model(&chat).Update("deleted_by_buyer", false)
		}
	}

	// 4. Create Message if content provided
	if input.Content != "" {
		// Reset seller deleted flag if they are receiving a message
		if chat.DeletedBySeller {
			database.DB.Model(&chat).Update("deleted_by_seller", false)
		}

		message := models.Message{
			ChatID:     chat.ID,
			SenderID:   userID,
			SenderRole: "BUYER",
			Content:    input.Content,
		}
		database.DB.Create(&message)

		// Send real-time notification via WebSocket
		hub := realtime.GetHub()
		notification, _ := json.Marshal(gin.H{
			"type":    "new_message",
			"chat_id": chat.ID,
			"message": message,
		})
		hub.SendToUser(seller.UserID, notification)

		// Send FCM push notification to seller
		var sellerUser models.User
		if err := database.DB.First(&sellerUser, seller.UserID).Error; err == nil && sellerUser.FCMToken != "" {
			_ = realtime.SendFCMNotification(
				sellerUser.FCMToken,
				"New Message",
				input.Content,
				map[string]string{
					"chat_id": strconv.Itoa(int(chat.ID)),
					"type":    "chat_message",
				},
			)
		}
	}

	// Populate transient fields for response
	chat.ProductName = product.Title
	url, _ := services.GenerateSignedURL(product.ImageName)
	chat.ProductImage = url

	c.JSON(http.StatusOK, chat)
}

func GetChats(c *gin.Context) {
	userID := c.GetUint("userID")

	// Find seller ID if user is a seller
	var sellerID uint
	var seller models.Seller
	if err := database.DB.Where("user_id = ?", userID).First(&seller).Error; err == nil {
		sellerID = seller.ID
	}

	var chats []models.Chat
	// Fetch chats where user is buyer and NOT deleted by buyer
	// OR user is seller and NOT deleted by seller
	query := database.DB.Where("(buyer_id = ? AND deleted_by_buyer = false)", userID)
	if sellerID != 0 {
		query = query.Or("(seller_id = ? AND deleted_by_seller = false)", sellerID)
	}

	if err := query.Find(&chats).Error; err != nil {
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
	isBuyer := false
	isParticipant := false
	if chat.BuyerID == userID {
		isBuyer = true
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
	query := database.DB.Where("chat_id = ?", chatID)
	if isBuyer {
		query = query.Where("deleted_by_buyer = false")
	} else {
		query = query.Where("deleted_by_seller = false")
	}

	if err := query.Find(&messages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch messages"})
		return
	}

	c.JSON(http.StatusOK, messages)
}

func DeleteChat(c *gin.Context) {
	userID := c.GetUint("userID")
	chatID := c.Param("chat_id")

	var chat models.Chat
	if err := database.DB.First(&chat, chatID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chat not found"})
		return
	}

	isBuyer := false
	authorized := false

	// Determine if user is buyer or seller
	if chat.BuyerID == userID {
		isBuyer = true
		chat.DeletedByBuyer = true
		authorized = true
	} else {
		var seller models.Seller
		if err := database.DB.Where("id = ?", chat.SellerID).First(&seller).Error; err == nil {
			if seller.UserID == userID {
				chat.DeletedBySeller = true
				authorized = true
			}
		}
	}

	if !authorized {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized"})
		return
	}

	// Mark all messages as deleted for this user
	if err := chat.MarkAllMessagesAsDeletedForUser(database.DB, userID, isBuyer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to flag messages"})
		return
	}

	if err := database.DB.Save(&chat).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete chat"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Chat deleted"})
}
