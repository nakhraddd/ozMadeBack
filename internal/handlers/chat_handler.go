package handlers

import (
	"net/http"
	"ozMadeBack/internal/database"
	"ozMadeBack/internal/models"

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
