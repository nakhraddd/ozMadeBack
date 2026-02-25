package handlers

import (
	"net/http"
	"ozMadeBack/internal/database"
	"ozMadeBack/internal/models"
	"ozMadeBack/internal/services"
	"strconv"

	"github.com/gin-gonic/gin"
)

func GetProducts(c *gin.Context) {
	var products []models.Product
	query := database.DB.Model(&models.Product{})

	if typeFilter := c.Query("type"); typeFilter != "" {
		query = query.Where("type = ?", typeFilter)
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	query.Limit(limit).Offset(offset).Find(&products)

	// Fetch seller names
	var sellerIDs []uint
	for _, p := range products {
		sellerIDs = append(sellerIDs, p.SellerID)
	}

	if len(sellerIDs) > 0 {
		var sellers []models.Seller
		database.DB.Preload("User").Where("id IN ?", sellerIDs).Find(&sellers)

		sellerMap := make(map[uint]string)
		for _, s := range sellers {
			sellerMap[s.ID] = s.User.Email
		}

		for i := range products {
			products[i].SellerName = sellerMap[products[i].SellerID]
		}
	}

	for i := range products {
		url, _ := services.GenerateSignedURL(products[i].ImageName)
		products[i].ImageName = url
	}

	c.JSON(http.StatusOK, products)
}

func GetProduct(c *gin.Context) {
	id := c.Param("id")
	var product models.Product
	if err := database.DB.Preload("Comments").First(&product, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	url, _ := services.GenerateSignedURL(product.ImageName)
	product.ImageName = url

	// Fetch seller name
	var seller models.Seller
	if err := database.DB.Preload("User").First(&seller, product.SellerID).Error; err == nil {
		product.SellerName = seller.User.Email
	}

	c.JSON(http.StatusOK, product)
}

func ViewProduct(c *gin.Context) {
	id := c.Param("id")
	var product models.Product
	if err := database.DB.First(&product, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	product.ViewCount++
	database.DB.Save(&product)
	database.RDB.ZIncrBy(c.Request.Context(), "trending_products", 1, id)

	c.Status(http.StatusOK)
}

func GetTrendingProducts(c *gin.Context) {
	ids, err := database.RDB.ZRevRange(c.Request.Context(), "trending_products", 0, 19).Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch trending products"})
		return
	}

	var products []models.Product
	if len(ids) > 0 {
		database.DB.Where("id IN ?", ids).Find(&products)

		// Fetch seller names
		var sellerIDs []uint
		for _, p := range products {
			sellerIDs = append(sellerIDs, p.SellerID)
		}

		if len(sellerIDs) > 0 {
			var sellers []models.Seller
			database.DB.Preload("User").Where("id IN ?", sellerIDs).Find(&sellers)

			sellerMap := make(map[uint]string)
			for _, s := range sellers {
				sellerMap[s.ID] = s.User.Email
			}

			for i := range products {
				products[i].SellerName = sellerMap[products[i].SellerID]
			}
		}

		for i := range products {
			url, _ := services.GenerateSignedURL(products[i].ImageName)
			products[i].ImageName = url
		}
	} else {
		products = []models.Product{}
	}

	c.JSON(http.StatusOK, products)
}
