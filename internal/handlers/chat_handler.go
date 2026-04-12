package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"ozMadeBack/internal/database"
	"ozMadeBack/internal/dto"
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
		c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "Chat not found"})
		return
	}

	senderRole := ""
	var recipientID uint

	if chat.BuyerID == userID {
		senderRole = "BUYER"
		var seller models.Seller
		if err := database.DB.First(&seller, chat.SellerID).Error; err == nil {
			recipientID = seller.UserID
		}
	} else {
		var seller models.Seller
		if err := database.DB.Where("id = ?", chat.SellerID).First(&seller).Error; err == nil {
			if seller.UserID == userID {
				senderRole = "SELLER"
				recipientID = chat.BuyerID
			}
		}
	}

	if senderRole == "" {
		c.JSON(http.StatusForbidden, dto.ErrorResponse{Error: "Unauthorized"})
		return
	}

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

	mediaUrl := ""
	mediaType := ""
	content := c.PostForm("content")

	file, err := c.FormFile("media")
	if err == nil {
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

		if services.GCS != nil {
			objectName := fmt.Sprintf("chats/%d/%s%s", chat.ID, uuid.New().String(), ext)
			mediaUrl = objectName
		}
	} else {
		var input dto.SendMessageInput
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
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "Failed to send message"})
		return
	}

	if message.MediaUrl != "" {
		message.MediaUrl, _ = services.GenerateSignedURLForChat(message.MediaUrl)
	}

	hub := realtime.GetHub()
	notificationPayload, _ := json.Marshal(gin.H{
		"type":    "new_message",
		"chat_id": chat.ID,
		"message": message,
	})
	hub.SendToUser(recipientID, notificationPayload)

	pushContent := content
	if pushContent == "" && mediaType != "" {
		pushContent = "Sent a " + mediaType
	}

	_ = services.CreateNotification(
		recipientID,
		"New Message",
		pushContent,
		"chat_message",
		nil,
		map[string]string{
			"chat_id": strconv.Itoa(int(chat.ID)),
		},
	)

	c.JSON(http.StatusCreated, message)
}

func InitiateChat(c *gin.Context) {
	userID := c.GetUint("userID")
	var input dto.InitiateChatInput

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
		return
	}

	var product models.Product
	if err := database.DB.First(&product, input.ProductID).Error; err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "Product not found"})
		return
	}

	var seller models.Seller
	if err := database.DB.Preload("User").Where("id = ?", product.SellerID).First(&seller).Error; err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "Seller not found"})
		return
	}

	if seller.UserID == userID {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "Cannot chat with yourself"})
		return
	}

	var chat models.Chat
	err := database.DB.Where("buyer_id = ? AND seller_id = ? AND product_id = ?", userID, seller.ID, product.ID).First(&chat).Error

	if err != nil {
		chat = models.Chat{
			BuyerID:   userID,
			SellerID:  seller.ID,
			ProductID: product.ID,
		}
		if err := database.DB.Create(&chat).Error; err != nil {
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "Failed to create chat"})
			return
		}
	} else {
		if chat.DeletedByBuyer {
			database.DB.Model(&chat).Update("deleted_by_buyer", false)
			chat.DeletedByBuyer = false
		}
	}

	if input.Content != "" {
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

		hub := realtime.GetHub()
		notificationPayload, _ := json.Marshal(gin.H{
			"type":    "new_message",
			"chat_id": chat.ID,
			"message": message,
		})
		hub.SendToUser(seller.UserID, notificationPayload)

		_ = services.CreateNotification(
			seller.UserID,
			"New Message",
			input.Content,
			"chat_message",
			nil,
			map[string]string{
				"chat_id": strconv.Itoa(int(chat.ID)),
			},
		)
	}

	var buyer models.User
	database.DB.First(&buyer, userID)

	chat.ProductName = product.Title

	displayImage := product.ImageName
	if displayImage == "" && len(product.Images) > 0 {
		displayImage = product.Images[0]
	}
	if displayImage != "" {
		url, _ := services.GenerateSignedURL(displayImage)
		chat.ProductImage = url
	}

	chat.SellerName = resolveSellerDisplayNameForChat(seller)
	chat.BuyerName = buyer.Name
	chat.SellerPhoto, _ = services.GenerateSignedURLForSeller(seller.PhotoURL)
	chat.BuyerPhoto, _ = services.GenerateSignedURLForUser(buyer.PhotoUrl)
	chat.PhoneNumber = seller.User.PhoneNumber

	c.JSON(http.StatusOK, chat)
}

func GetChats(c *gin.Context) {
	userID := c.GetUint("userID")
	role := c.Query("role")

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
		q := "(buyer_id = ? AND deleted_by_buyer = false)"
		args := []interface{}{userID}
		if sellerID != 0 {
			q += " OR (seller_id = ? AND deleted_by_seller = false)"
			args = append(args, sellerID)
		}
		query = query.Where(q, args...)
	}

	if err := query.Find(&chats).Error; err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "Failed to fetch chats"})
		return
	}

	for i := range chats {
		if chats[i].ProductID != 0 {
			var product models.Product
			if err := database.DB.First(&product, chats[i].ProductID).Error; err == nil {
				chats[i].ProductName = product.Title

				displayImage := product.ImageName
				if displayImage == "" && len(product.Images) > 0 {
					displayImage = product.Images[0]
				}
				if displayImage != "" {
					url, _ := services.GenerateSignedURL(displayImage)
					chats[i].ProductImage = url
				}
			}
		}

		var s models.Seller
		var b models.User
		database.DB.Preload("User").First(&s, chats[i].SellerID)
		database.DB.First(&b, chats[i].BuyerID)

		chats[i].SellerName = resolveSellerDisplayNameForChat(s)
		chats[i].BuyerName = b.Name
		chats[i].SellerPhoto, _ = services.GenerateSignedURLForSeller(s.PhotoURL)
		chats[i].BuyerPhoto, _ = services.GenerateSignedURLForUser(b.PhotoUrl)

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
		c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "Chat not found"})
		return
	}

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
		c.JSON(http.StatusForbidden, dto.ErrorResponse{Error: "Unauthorized"})
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
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "Failed to fetch messages"})
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
		c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "Chat not found"})
		return
	}

	isBuyer := false
	authorized := false

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
		c.JSON(http.StatusForbidden, dto.ErrorResponse{Error: "Unauthorized"})
		return
	}

	if err := chat.MarkAllMessagesAsDeletedForUser(database.DB, userID, isBuyer); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "Failed to flag messages"})
		return
	}

	if err := database.DB.Save(&chat).Error; err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "Failed to delete chat"})
		return
	}

	c.JSON(http.StatusOK, dto.MessageResponse{Message: "Chat deleted"})
}

func GetUploadURL(c *gin.Context) {
	chatIDStr := c.Query("chat_id")
	fileName := c.Query("file_name")
	contentType := c.Query("content_type")

	if chatIDStr == "" || fileName == "" || contentType == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "chat_id, file_name, and content_type are required"})
		return
	}

	if services.GCS == nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "GCS service not initialized"})
		return
	}

	ext := filepath.Ext(fileName)
	objectName := fmt.Sprintf("chats/%s/%s%s", chatIDStr, uuid.New().String(), ext)

	url, err := services.GCS.GenerateSignedURL(objectName, "PUT", 15*time.Minute, contentType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "Failed to generate upload URL"})
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
