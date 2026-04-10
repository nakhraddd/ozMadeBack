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
		name, _ := firebaseToken.Claims["name"].(string)
		photoURL, _ := firebaseToken.Claims["picture"].(string)

		user = models.User{
			FirebaseUID: firebaseToken.UID,
			PhoneNumber: phoneNumber,
			Name:        name,
			PhotoUrl:    photoURL,
		}

		if err := database.DB.Create(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
			return
		}
	} else {
		// Update name or photo if they changed in Firebase/are missing
		updated := false
		if name, ok := firebaseToken.Claims["name"].(string); ok && user.Name == "" {
			user.Name = name
			updated = true
		}
		if photoURL, ok := firebaseToken.Claims["picture"].(string); ok && user.PhotoUrl == "" {
			user.PhotoUrl = photoURL
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
