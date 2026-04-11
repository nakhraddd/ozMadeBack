package handlers

import (
	"fmt"
	"net/http"
	"ozMadeBack/internal/database"
	"ozMadeBack/internal/models"
	"ozMadeBack/internal/services"
	"path/filepath"
	"strconv"
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

	var input struct {
		Name       *string  `json:"name"`
		Address    *string  `json:"address"`
		AddressLat *float64 `json:"address_lat"`
		AddressLng *float64 `json:"address_lng"`
		PhotoUrl   *string  `json:"photo_url"`
		FCMToken   *string  `json:"fcm_token"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := make(map[string]interface{})
	if input.Name != nil {
		updates["name"] = *input.Name
	}
	if input.Address != nil {
		updates["address"] = *input.Address
	}
	if input.PhotoUrl != nil {
		updates["photo_url"] = *input.PhotoUrl
	}
	if input.FCMToken != nil {
		updates["fcm_token"] = *input.FCMToken
	}

	if input.AddressLat != nil && input.AddressLng != nil {
		updates["address_lat"] = *input.AddressLat
		updates["address_lng"] = *input.AddressLng
	} else if (input.AddressLat == nil) != (input.AddressLng == nil) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "address_lat and address_lng must be provided together"})
		return
	}

	if len(updates) > 0 {
		if err := database.DB.Model(&user).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
			return
		}
	}

	// Fetch updated user to return
	database.DB.First(&user, userID)
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
