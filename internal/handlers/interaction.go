package handlers

import (
	"net/http"
	"ozMadeBack/internal/database"
	"ozMadeBack/internal/dto"
	"ozMadeBack/internal/models"
	"ozMadeBack/internal/services"
	"strconv"

	"github.com/gin-gonic/gin"
)

func GetProductReviews(c *gin.Context) {
	productIDStr := c.Param("id")
	productID, err := strconv.ParseUint(productIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "Invalid product ID"})
		return
	}

	var product models.Product
	if err := database.DB.First(&product, productID).Error; err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "Product not found"})
		return
	}

	var comments []models.Comment
	database.DB.Preload("User").
		Where("product_id = ?", productID).
		Order("created_at desc").Find(&comments)

	reviewDtos := make([]dto.ReviewDto, 0, len(comments))
	for _, comment := range comments {
		name := comment.User.Name
		if name == "" {
			name = comment.User.PhoneNumber
		}
		if name == "" {
			name = "Anonymous"
		}

		photo, _ := services.GenerateSignedURLForUser(comment.User.PhotoUrl)

		reviewDtos = append(reviewDtos, dto.ReviewDto{
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

	response := dto.ProductReviewsResponse{
		Reviews: reviewDtos,
	}
	response.Summary.ProductID = uint(productID)
	response.Summary.AverageRating = product.AverageRating
	response.Summary.RatingsCount = ratingsCount
	response.Summary.ReviewsCount = reviewsCount

	c.JSON(http.StatusOK, response)
}

func GetSellerReviews(c *gin.Context) {
	sellerIDStr := c.Param("id")
	sellerID, err := strconv.ParseUint(sellerIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "Invalid seller ID"})
		return
	}

	var seller models.Seller
	if err := database.DB.Preload("User").First(&seller, sellerID).Error; err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "Seller not found"})
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

	reviewDtos := make([]dto.ReviewDto, 0, len(comments))
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

		reviewDtos = append(reviewDtos, dto.ReviewDto{
			ID:           comment.ID,
			UserID:       comment.UserID,
			UserName:     name,
			UserPhoto:    photo,
			ProductID:    comment.ProductID,
			ProductTitle: comment.Product.Title,
			Rating:       comment.Rating,
			CreatedAt:    comment.CreatedAt,
			Text:         comment.Text,
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

	response := dto.SellerReviewsResponse{
		Reviews: reviewDtos,
	}
	response.Header.SellerID = sellerID
	response.Header.SellerName = sellerName
	response.Header.ReviewsCount = reviewsWithTextCount
	response.Header.AverageRating = avgRating
	response.Header.RatingsCount = len(comments)

	c.JSON(http.StatusOK, response)
}

func PostComment(c *gin.Context) {
	userID := c.GetUint("userID")
	productIDStr := c.Param("id")
	productID, err := strconv.ParseUint(productIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "Invalid product ID"})
		return
	}

	var input dto.PostCommentInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
		return
	}

	comment := models.Comment{
		UserID:    userID,
		ProductID: uint(productID),
		Rating:    input.Rating,
		Text:      input.Text,
	}
	if err := database.DB.Create(&comment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "Failed to post comment"})
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
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "Invalid product ID"})
		return
	}

	var input dto.ReportProductInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
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
