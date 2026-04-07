package handlers

import (
	"net/http"
	"ozMadeBack/internal/database"
	"ozMadeBack/internal/models"

	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
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
		email, _ := firebaseToken.Claims["email"].(string)
		name, _ := firebaseToken.Claims["name"].(string)

		user = models.User{
			FirebaseUID: firebaseToken.UID,
			PhoneNumber: phoneNumber,
			Email:       email,
			Name:        name,
		}

		if err := database.DB.Create(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
			return
		}
	} else {
		// Update email or name if they changed in Firebase/are missing
		updated := false
		if email, ok := firebaseToken.Claims["email"].(string); ok && user.Email == "" {
			user.Email = email
			updated = true
		}
		if name, ok := firebaseToken.Claims["name"].(string); ok && user.Name == "" {
			user.Name = name
			updated = true
		}
		if updated {
			database.DB.Save(&user)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id": user.ID,
		"profile": user,
	})
}
