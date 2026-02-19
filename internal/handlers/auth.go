package handlers

import (
	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
	"net/http"
	"ozMadeBack/internal/database"
	"ozMadeBack/internal/models"
)

func SyncUser(c *gin.Context) {
	val, exists := c.Get("firebaseToken")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Firebase token not found in context"})
		return
	}

	firebaseToken, ok := val.(*auth.Token)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid token type"})
		return
	}

	var user models.User
	result := database.DB.Where("firebase_uid = ?", firebaseToken.UID).First(&user)

	if result.Error != nil {
		phoneNumber, _ := firebaseToken.Claims["phone_number"].(string)

		user = models.User{
			FirebaseUID: firebaseToken.UID,
			PhoneNumber: phoneNumber,
		}

		if err := database.DB.Create(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id": user.ID,
		"profile": user,
	})
}
