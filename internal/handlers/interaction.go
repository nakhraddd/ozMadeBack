package handlers

import (
	"net/http"
	"ozMadeBack/internal/database"
	"ozMadeBack/internal/models"
	"strconv"

	"github.com/gin-gonic/gin"
)

func PostComment(c *gin.Context) {
	userID, _ := c.Get("user_id")
	productIDStr := c.Param("id")
	productID, err := strconv.ParseUint(productIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	var input struct {
		Rating int    `json:"rating"`
		Text   string `json:"text"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	comment := models.Comment{
		UserID:    userID.(uint),
		ProductID: uint(productID),
		Rating:    float64(input.Rating),
		Text:      input.Text,
	}
	database.DB.Create(&comment)

	go updateAverageRating(uint(productID))

	c.JSON(http.StatusCreated, comment)
}

func ReportProduct(c *gin.Context) {
	userID, _ := c.Get("user_id")
	productIDStr := c.Param("id")
	productID, err := strconv.ParseUint(productIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	var input struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	report := models.Report{
		UserID:    userID.(uint),
		ProductID: uint(productID),
		Reason:    input.Reason,
	}
	database.DB.Create(&report)

	c.Status(http.StatusCreated)
}

func updateAverageRating(productID uint) {
	var comments []models.Comment
	database.DB.Where("product_id = ?", productID).Find(&comments)

	var totalRating float64
	for _, c := range comments {
		totalRating += c.Rating
	}

	avgRating := 0.0
	if len(comments) > 0 {
		avgRating = totalRating / float64(len(comments))
	}

	var product models.Product
	database.DB.First(&product, productID)
	product.AverageRating = avgRating
	database.DB.Save(&product)
}
