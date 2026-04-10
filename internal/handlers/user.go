package handlers

import (
	"fmt"
	"net/http"
	"ozMadeBack/internal/database"
	"ozMadeBack/internal/models"
	"ozMadeBack/internal/services"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func GetProfile(c *gin.Context) {
	userID := c.GetUint("userID")
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if user.PhotoUrl != "" {
		user.PhotoUrl, _ = services.GenerateSignedURLForUser(user.PhotoUrl)
	}

	c.JSON(http.StatusOK, user)
}

func UpdateProfile(c *gin.Context) {
	userID := c.GetUint("userID")

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Handle multipart form for photo upload
	contentType := c.GetHeader("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		// Update fields from form
		if name := c.PostForm("name"); name != "" {
			user.Name = name
		}
		if address := c.PostForm("address"); address != "" {
			user.Address = address
		}
		if phoneNumber := c.PostForm("phone_number"); phoneNumber != "" {
			user.PhoneNumber = phoneNumber
		}
		if fcmToken := c.PostForm("fcm_token"); fcmToken != "" {
			user.FCMToken = fcmToken
		}

		// Handle photo upload
		file, err := c.FormFile("photo")
		if err == nil && services.GCS != nil {
			ext := filepath.Ext(file.Filename)
			objectName := fmt.Sprintf("users/%d/%s%s", user.ID, uuid.New().String(), ext)

			// Open the file
			f, _ := file.Open()
			defer f.Close()

			// Upload to GCS
			wc := services.GCS.Client.Bucket(services.GCS.BucketName).Object(objectName).NewWriter(c.Request.Context())
			// io.Copy(wc, f) then wc.Close()
			_ = wc

			user.PhotoUrl = objectName
		}
	} else {
		// Handle JSON update
		var input map[string]interface{}
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Use a map to update only provided fields
		// Filter out sensitive fields like ID or FirebaseUID if needed
		delete(input, "id")
		delete(input, "firebase_uid")

		if err := database.DB.Model(&user).Updates(input).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
			return
		}

		// Refresh user from DB to get the updated values
		database.DB.First(&user, userID)
	}

	if err := database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save profile"})
		return
	}

	if user.PhotoUrl != "" {
		user.PhotoUrl, _ = services.GenerateSignedURLForUser(user.PhotoUrl)
	}

	c.JSON(http.StatusOK, user)
}

func UpdateFCMToken(c *gin.Context) {
	userID := c.GetUint("userID")
	var input struct {
		FCMToken string `json:"fcm_token"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	user.FCMToken = input.FCMToken
	database.DB.Save(&user)
	c.JSON(http.StatusOK, gin.H{"message": "FCM token updated"})
}

func ToggleFavorite(c *gin.Context) {
	userID := c.GetUint("userID")
	productIDStr := c.Param("id")
	productID, err := strconv.ParseUint(productIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	var favorite models.Favorite
	if err := database.DB.Where("user_id = ? AND product_id = ?", userID, productID).First(&favorite).Error; err != nil {
		newFavorite := models.Favorite{UserID: userID, ProductID: uint(productID)}
		database.DB.Create(&newFavorite)
		c.JSON(http.StatusOK, gin.H{"status": "added"})
	} else {
		database.DB.Delete(&favorite)
		c.JSON(http.StatusOK, gin.H{"status": "removed"})
	}
}

func GetFavorites(c *gin.Context) {
	userID := c.GetUint("userID")
	var favorites []models.Favorite
	database.DB.Where("user_id = ?", userID).Find(&favorites)

	var productIDs []uint
	for _, f := range favorites {
		productIDs = append(productIDs, f.ProductID)
	}

	var products []models.Product
	database.DB.Where("id IN ?", productIDs).Find(&products)

	for i := range products {
		if products[i].ImageName != "" {
			url, _ := services.GenerateSignedURL(products[i].ImageName)
			products[i].ImageName = url
		}
		for j, imgName := range products[i].Images {
			if imgName != "" {
				gUrl, _ := services.GenerateSignedURL(imgName)
				products[i].Images[j] = gUrl
			}
		}
	}

	c.JSON(http.StatusOK, products)
}

func GetProfileUploadURL(c *gin.Context) {
	userID := c.GetUint("userID")
	fileName := c.Query("file_name")
	contentType := c.Query("content_type")

	if fileName == "" || contentType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file_name and content_type are required"})
		return
	}

	if services.GCS == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "GCS service not initialized"})
		return
	}

	ext := filepath.Ext(fileName)
	objectName := fmt.Sprintf("users/%d/%s%s", userID, uuid.New().String(), ext)

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
