package handlers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"ozMadeBack/internal/database"
	"ozMadeBack/internal/models"
	"ozMadeBack/internal/services"
	"strconv"
)

func GetProfile(c *gin.Context) {
	userID, _ := c.Get("user_id")
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	c.JSON(http.StatusOK, user)
}

func UpdateProfile(c *gin.Context) {
	userID, _ := c.Get("user_id")
	var input struct {
		Email   string `json:"email"`
		Address string `json:"address"`
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

	if input.Email != "" {
		user.Email = input.Email
	}
	if input.Address != "" {
		user.Address = input.Address
	}

	database.DB.Save(&user)
	c.JSON(http.StatusOK, user)
}

func ToggleFavorite(c *gin.Context) {
	userID, _ := c.Get("user_id")
	productIDStr := c.Param("id")
	productID, err := strconv.ParseUint(productIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	var favorite models.Favorite
	if err := database.DB.Where("user_id = ? AND product_id = ?", userID, productID).First(&favorite).Error; err != nil {
		newFavorite := models.Favorite{UserID: userID.(uint), ProductID: uint(productID)}
		database.DB.Create(&newFavorite)
		c.JSON(http.StatusOK, gin.H{"status": "added"})
	} else {
		database.DB.Delete(&favorite)
		c.JSON(http.StatusOK, gin.H{"status": "removed"})
	}
}

func GetFavorites(c *gin.Context) {
	userID, _ := c.Get("user_id")
	var favorites []models.Favorite
	database.DB.Where("user_id = ?", userID).Find(&favorites)

	var productIDs []uint
	for _, f := range favorites {
		productIDs = append(productIDs, f.ProductID)
	}

	var products []models.Product
	database.DB.Where("id IN ?", productIDs).Find(&products)

	for i := range products {
		url, _ := services.GenerateSignedURL(products[i].ImageName)
		products[i].ImageName = url
	}

	c.JSON(http.StatusOK, products)
}

func GetOrders(c *gin.Context) {
	userID, _ := c.Get("user_id")
	var orders []models.Order
	database.DB.Where("user_id = ?", userID).Find(&orders)
	c.JSON(http.StatusOK, orders)
}
