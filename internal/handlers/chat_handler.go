package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"ozMadeBack/internal/database"
	"ozMadeBack/internal/models"
	"ozMadeBack/internal/services"
	"ozMadeBack/pkg/realtime"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func SendMessage(c *gin.Context) {
	userID := c.GetUint("userID")
	chatID := c.Param("chat_id")

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

	// Handle file upload if present
	mediaUrl := ""
	mediaType := ""
	content := c.PostForm("content")

	file, err := c.FormFile("media")
	if err == nil {
		// Detect media type
		ext := strings.ToLower(filepath.Ext(file.Filename))
		switch ext {
		case ".jpg", ".jpeg", ".png", ".gif", ".webp":
			mediaType = "photo"
		case ".mp3", ".wav", ".m4a", ".ogg":
			mediaType = "audio"
		case ".mp4", ".mov", ".avi", ".mkv":
			mediaType = "video"
		default:
			mediaType = "file"
		}

		// Upload to GCS
		if services.GCS != nil {
			objectName := fmt.Sprintf("chats/%d/%s%s", chat.ID, uuid.New().String(), ext)
			f, _ := file.Open()
			defer f.Close()

			wc := services.GCS.Client.Bucket(services.GCS.BucketName).Object(objectName).NewWriter(c.Request.Context())
			// io.Copy(wc, f) then wc.Close()
			_ = wc

			mediaUrl = objectName
		}
	} else {
		// If not a form-data request, try JSON
		var input struct {
			Content   string `json:"content"`
			MediaUrl  string `json:"media_url"`
			MediaType string `json:"media_type"`
		}
		if err := c.ShouldBindJSON(&input); err == nil {
			content = input.Content
			mediaUrl = input.MediaUrl
			mediaType = input.MediaType
		}
	}

	message := models.Message{
		ChatID:     chat.ID,
		SenderID:   userID,
		SenderRole: senderRole,
		Content:    content,
		MediaUrl:   mediaUrl,
		MediaType:  mediaType,
	}

	if err := database.DB.Create(&message).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send message"})
		return
	}

	// Sign media URL for response
	if message.MediaUrl != "" {
		message.MediaUrl, _ = services.GenerateSignedURLForChat(message.MediaUrl)
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
		pushContent := content
		if pushContent == "" && mediaType != "" {
			pushContent = "Sent a " + mediaType
		}
		_ = realtime.SendFCMNotification(
			recipient.FCMToken,
			"New Message",
			pushContent,
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
	if err := database.DB.Preload("User").Where("id = ?", product.SellerID).First(&seller).Error; err != nil {
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
			chat.DeletedByBuyer = false
		}
	}

	// 4. Create Message if content provided
	if input.Content != "" {
		// Reset seller deleted flag if they are receiving a message
		if chat.DeletedBySeller {
			database.DB.Model(&chat).Update("deleted_by_seller", false)
			chat.DeletedBySeller = false
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
		if seller.User.FCMToken != "" {
			_ = realtime.SendFCMNotification(
				seller.User.FCMToken,
				"New Message",
				input.Content,
				map[string]string{
					"chat_id": strconv.Itoa(int(chat.ID)),
					"type":    "chat_message",
				},
			)
		}
	}

	// Fetch Buyer for the response names
	var buyer models.User
	database.DB.First(&buyer, userID)

	// Populate transient fields for response
	chat.ProductName = product.Title
	url, _ := services.GenerateSignedURL(product.ImageName)
	chat.ProductImage = url
	chat.SellerName = resolveSellerDisplayNameForChat(seller)
	chat.BuyerName = buyer.Name
	chat.SellerPhoto, _ = services.GenerateSignedURLForSeller(seller.PhotoURL)
	chat.BuyerPhoto, _ = services.GenerateSignedURLForUser(buyer.PhotoUrl)
	chat.PhoneNumber = seller.User.PhoneNumber // As initiator (buyer), phone number is the seller's

	c.JSON(http.StatusOK, chat)
}

func GetChats(c *gin.Context) {
	userID := c.GetUint("userID")
	role := c.Query("role") // "buyer" or "seller"

	// Find seller ID if user is a seller
	var sellerID uint
	var seller models.Seller
	if err := database.DB.Where("user_id = ?", userID).First(&seller).Error; err == nil {
		sellerID = seller.ID
	}

	var chats []models.Chat
	query := database.DB.Model(&models.Chat{})

	if role == "buyer" {
		query = query.Where("buyer_id = ? AND deleted_by_buyer = false", userID)
	} else if role == "seller" && sellerID != 0 {
		query = query.Where("seller_id = ? AND deleted_by_seller = false", sellerID)
	} else {
		// Default behavior: show both if role not specified
		q := "(buyer_id = ? AND deleted_by_buyer = false)"
		args := []interface{}{userID}
		if sellerID != 0 {
			q += " OR (seller_id = ? AND deleted_by_seller = false)"
			args = append(args, sellerID)
		}
		query = query.Where(q, args...)
	}

	if err := query.Find(&chats).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch chats"})
		return
	}

	for i := range chats {
		// Populate Product info
		if chats[i].ProductID != 0 {
			var product models.Product
			if err := database.DB.First(&product, chats[i].ProductID).Error; err == nil {
				chats[i].ProductName = product.Title
				url, _ := services.GenerateSignedURL(product.ImageName)
				chats[i].ProductImage = url
			}
		}

		// Populate Names, Photos and Phone Number
		var s models.Seller
		var b models.User
		database.DB.Preload("User").First(&s, chats[i].SellerID)
		database.DB.First(&b, chats[i].BuyerID)

		chats[i].SellerName = resolveSellerDisplayNameForChat(s)
		chats[i].BuyerName = b.Name
		chats[i].SellerPhoto, _ = services.GenerateSignedURLForSeller(s.PhotoURL)
		chats[i].BuyerPhoto, _ = services.GenerateSignedURLForUser(b.PhotoUrl)

		// Return the phone number of the "other" party
		if userID == chats[i].BuyerID {
			chats[i].PhoneNumber = s.User.PhoneNumber
		} else {
			chats[i].PhoneNumber = b.PhoneNumber
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

	for i := range messages {
		if messages[i].MediaUrl != "" {
			messages[i].MediaUrl, _ = services.GenerateSignedURLForChat(messages[i].MediaUrl)
		}
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

	// Mark all current messages as deleted for this user
	if err := chat.MarkAllMessagesAsDeletedForUser(database.DB, userID, isBuyer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to flag messages"})
		return
	}

	// Also flag the chat as deleted for this user
	if err := database.DB.Save(&chat).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete chat"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Chat deleted"})
}

func GetUploadURL(c *gin.Context) {
	chatIDStr := c.Query("chat_id")
	fileName := c.Query("file_name")
	contentType := c.Query("content_type")

	if chatIDStr == "" || fileName == "" || contentType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "chat_id, file_name, and content_type are required"})
		return
	}

	if services.GCS == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "GCS service not initialized"})
		return
	}

	ext := filepath.Ext(fileName)
	objectName := fmt.Sprintf("chats/%s/%s%s", chatIDStr, uuid.New().String(), ext)

	url, err := services.GCS.GenerateSignedURL(objectName, "PUT", 15*time.Minute, contentType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate upload URL"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"upload_url":  url,
		"object_name": objectName,
	})
}

func resolveSellerDisplayNameForChat(seller models.Seller) string {
	if seller.DisplayName != "" {
		return seller.DisplayName
	}
	if seller.User.Name != "" {
		return seller.User.Name
	}
	return "Seller"
}
