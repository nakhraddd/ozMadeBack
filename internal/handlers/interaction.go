package handlers

import (
	"net/http"
	"ozMadeBack/internal/database"
	"ozMadeBack/internal/models"
	"ozMadeBack/internal/services"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type ReviewDto struct {
	ID        uint      `json:"id"`
	UserID    uint      `json:"user_id"`
	UserName  string    `json:"user_name"`
	UserPhoto string    `json:"user_photo"`
	Rating    float64   `json:"rating"`
	CreatedAt time.Time `json:"created_at"`
	Text      string    `json:"text"`
}

func GetProductReviews(c *gin.Context) {
	productIDStr := c.Param("id")
	productID, err := strconv.ParseUint(productIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	var product models.Product
	if err := database.DB.First(&product, productID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	var comments []models.Comment
	database.DB.Preload("User").
		Where("product_id = ?", productID).
		Order("created_at desc").Find(&comments)

	reviewDtos := make([]ReviewDto, 0, len(comments))
	for _, comment := range comments {
		name := comment.User.Name
		if name == "" {
			name = comment.User.PhoneNumber
		}
		if name == "" {
			name = "Anonymous"
		}

		photo, _ := services.GenerateSignedURLForUser(comment.User.PhotoUrl)

		reviewDtos = append(reviewDtos, ReviewDto{
			ID:        comment.ID,
			UserID:    comment.UserID,
			UserName:  name,
			UserPhoto: photo,
			Rating:    comment.Rating,
			CreatedAt: comment.CreatedAt,
			Text:      comment.Text,
		})
	}

	var reviewsCount int64
	database.DB.Model(&models.Comment{}).Where("product_id = ? AND text != ''", productID).Count(&reviewsCount)

	var ratingsCount int64
	database.DB.Model(&models.Comment{}).Where("product_id = ?", productID).Count(&ratingsCount)

	c.JSON(http.StatusOK, gin.H{
		"summary": gin.H{
			"product_id":     productID,
			"average_rating": product.AverageRating,
			"ratings_count":  ratingsCount,
			"reviews_count":  reviewsCount,
		},
		"reviews": reviewDtos,
	})
}

func GetSellerReviews(c *gin.Context) {
	sellerIDStr := c.Param("id")
	sellerID, err := strconv.ParseUint(sellerIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid seller ID"})
		return
	}

	var seller models.Seller
	if err := database.DB.Preload("User").First(&seller, sellerID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Seller not found"})
		return
	}

	var productIDs []uint
	database.DB.Model(&models.Product{}).Where("seller_id = ?", sellerID).Pluck("id", &productIDs)

	var comments []models.Comment
	if len(productIDs) > 0 {
		database.DB.Preload("User").Preload("Product").
			Where("product_id IN ?", productIDs).
			Order("created_at desc").Find(&comments)
	}

	reviewDtos := make([]interface{}, 0, len(comments))
	var totalRating float64
	for _, comment := range comments {
		totalRating += comment.Rating

		name := comment.User.Name
		if name == "" {
			name = comment.User.PhoneNumber
		}
		if name == "" {
			name = "Anonymous"
		}
		photo, _ := services.GenerateSignedURLForUser(comment.User.PhotoUrl)

		reviewDtos = append(reviewDtos, gin.H{
			"id":            comment.ID,
			"user_id":       comment.UserID,
			"user_name":     name,
			"user_photo":    photo,
			"product_id":    comment.ProductID,
			"product_title": comment.Product.Title,
			"rating":        comment.Rating,
			"created_at":    comment.CreatedAt,
			"text":          comment.Text,
		})
	}

	avgRating := 0.0
	if len(comments) > 0 {
		avgRating = totalRating / float64(len(comments))
	}

	var reviewsWithTextCount int64
	if len(productIDs) > 0 {
		database.DB.Model(&models.Comment{}).Where("product_id IN ? AND text != ''", productIDs).Count(&reviewsWithTextCount)
	}

	sellerName := seller.User.Name
	if sellerName == "" {
		sellerName = seller.User.PhoneNumber
	}

	c.JSON(http.StatusOK, gin.H{
		"header": gin.H{
			"seller_id":      sellerID,
			"seller_name":    sellerName,
			"reviews_count":  reviewsWithTextCount,
			"average_rating": avgRating,
			"ratings_count":  len(comments),
		},
		"reviews": reviewDtos,
	})
}

func PostComment(c *gin.Context) {
	userID := c.GetUint("userID")
	productIDStr := c.Param("id")
	productID, err := strconv.ParseUint(productIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	var input struct {
		Rating float64 `json:"rating"`
		Text   string  `json:"text"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	comment := models.Comment{
		UserID:    userID,
		ProductID: uint(productID),
		Rating:    input.Rating,
		Text:      input.Text,
	}
	if err := database.DB.Create(&comment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to post comment"})
		return
	}

	go updateProductInteractionStats(uint(productID))

	c.JSON(http.StatusCreated, comment)
}

func ReportProduct(c *gin.Context) {
	userID := c.GetUint("userID")
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
		UserID:    userID,
		ProductID: uint(productID),
		Reason:    input.Reason,
	}
	database.DB.Create(&report)

	c.Status(http.StatusCreated)
}

func updateProductInteractionStats(productID uint) {
	var comments []models.Comment
	database.DB.Where("product_id = ?", productID).Find(&comments)

	var totalRating float64
	var reviewsCount int
	for _, c := range comments {
		totalRating += c.Rating
		if c.Text != "" {
			reviewsCount++
		}
	}

	ratingsCount := len(comments)
	avgRating := 0.0
	if ratingsCount > 0 {
		avgRating = totalRating / float64(ratingsCount)
	}

	database.DB.Model(&models.Product{}).Where("id = ?", productID).Updates(map[string]interface{}{
		"average_rating": avgRating,
		"ratings_count":  ratingsCount,
		"reviews_count":  reviewsCount,
	})
}
