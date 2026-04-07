package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"ozMadeBack/config"
	"ozMadeBack/internal/database"
	"ozMadeBack/internal/models"
	productservice "ozMadeBack/internal/service/product"
	"ozMadeBack/internal/services"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ProductResponse struct {
	models.Product
	Delivery  gin.H  `json:"delivery"`
	Seller    gin.H  `json:"seller"`
	ShareLink string `json:"share_link"`
}

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

	// Fetch sellers
	var sellerIDs []uint
	for _, p := range products {
		sellerIDs = append(sellerIDs, p.SellerID)
	}

	sellerMap := make(map[uint]models.Seller)
	if len(sellerIDs) > 0 {
		var sellers []models.Seller
		database.DB.Preload("User").Where("id IN ?", sellerIDs).Find(&sellers)
		for _, s := range sellers {
			sellerMap[s.ID] = s
		}
	}

	appLinkBase := config.GetEnv("APP_LINK_BASE_URL", "https://ozmade-applink.vercel.app")

	var response []ProductResponse
	for i := range products {
		// Generate signed URL for main image
		url, _ := services.GenerateSignedURL(products[i].ImageName)
		products[i].ImageName = url

		// Generate signed URLs for gallery images
		for j, imgName := range products[i].Images {
			gUrl, _ := services.GenerateSignedURL(imgName)
			products[i].Images[j] = gUrl
		}

		seller, exists := sellerMap[products[i].SellerID]
		delivery := gin.H{}
		sellerInfo := gin.H{}

		if exists {
			// Populate delivery info
			delivery = gin.H{
				"pickupEnabled":       seller.PickupEnabled,
				"pickupTime":          seller.PickupTime,
				"pickupAddress":       seller.PickupAddress,
				"freeDeliveryEnabled": seller.FreeDeliveryEnabled,
				"freeDeliveryText":    "Citywide", // Hardcoded as per example, or could be dynamic
				"intercityEnabled":    seller.IntercityEnabled,
				"deliveryCenterLat":   seller.DeliveryCenterLat,
				"deliveryCenterLng":   seller.DeliveryCenterLng,
				"deliveryRadiusKm":    seller.DeliveryRadiusKm,
			}

			// Populate seller info
			sellerInfo = gin.H{
				"id":      seller.ID,
				"name":    seller.User.Email, // Using email as name for now, or add Name field to User/Seller
				"address": "Almaty",          // Placeholder or needs to be added to Seller model if distinct
			}
			products[i].SellerName = seller.User.Email
		} else {
			// Default values if seller not found (shouldn't happen ideally)
			delivery = gin.H{
				"pickupEnabled":       false,
				"pickupTime":          nil,
				"pickupAddress":       nil,
				"freeDeliveryEnabled": false,
				"freeDeliveryText":    nil,
				"intercityEnabled":    false,
				"deliveryCenterLat":   nil,
				"deliveryCenterLng":   nil,
				"deliveryRadiusKm":    nil,
			}
			sellerInfo = gin.H{
				"id":      0,
				"name":    "Unknown",
				"address": "Unknown",
			}
		}

		response = append(response, ProductResponse{
			Product:   products[i],
			Delivery:  delivery,
			Seller:    sellerInfo,
			ShareLink: fmt.Sprintf("%s/products/%d", appLinkBase, products[i].ID),
		})
	}

	c.JSON(http.StatusOK, response)
}

func SearchProducts(c *gin.Context) {
	minCost, err := services.ParseSearchFloat(c.Query("min_cost"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid min_cost"})
		return
	}

	maxCost, err := services.ParseSearchFloat(c.Query("max_cost"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid max_cost"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	params := services.ProductSearchParams{
		Query:    c.Query("q"),
		Type:     c.Query("type"),
		Category: c.Query("category"),
		MinCost:  minCost,
		MaxCost:  maxCost,
		Limit:    limit,
		Offset:   (page - 1) * limit,
	}

	searchService := services.ProductSearch
	if searchService == nil {
		searchService = &services.ProductSearchService{}
	}

	productIDs, err := searchService.SearchProducts(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search products"})
		return
	}

	if len(productIDs) == 0 {
		c.JSON(http.StatusOK, []ProductResponse{})
		return
	}

	var products []models.Product
	if err := database.DB.Where("id IN ?", productIDs).Find(&products).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load products"})
		return
	}

	productsByID := make(map[uint]models.Product, len(products))
	for _, product := range products {
		productsByID[product.ID] = product
	}

	orderedProducts := make([]models.Product, 0, len(productIDs))
	for _, id := range productIDs {
		product, ok := productsByID[id]
		if !ok {
			continue
		}
		orderedProducts = append(orderedProducts, product)
	}

	c.JSON(http.StatusOK, buildProductResponses(orderedProducts))
}

func GetProduct(c *gin.Context) {
	id := c.Param("id")
	productID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	product, err := productservice.NewDefaultService().GetProduct(c.Request.Context(), uint(productID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch product"})
		return
	}

	if product.ID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	// Generate signed URL for main image
	url, _ := services.GenerateSignedURL(product.ImageName)
	product.ImageName = url

	// Generate signed URLs for gallery images
	for j, imgName := range product.Images {
		gUrl, _ := services.GenerateSignedURL(imgName)
		product.Images[j] = gUrl
	}

	// Fetch seller
	var seller models.Seller
	delivery := gin.H{}
	sellerInfo := gin.H{}

	if err := database.DB.Preload("User").First(&seller, product.SellerID).Error; err == nil {
		product.SellerName = seller.User.Email

		delivery = gin.H{
			"pickupEnabled":       seller.PickupEnabled,
			"pickupTime":          seller.PickupTime,
			"pickupAddress":       seller.PickupAddress,
			"freeDeliveryEnabled": seller.FreeDeliveryEnabled,
			"freeDeliveryText":    "Citywide",
			"intercityEnabled":    seller.IntercityEnabled,
			"deliveryCenterLat":   seller.DeliveryCenterLat,
			"deliveryCenterLng":   seller.DeliveryCenterLng,
			"deliveryRadiusKm":    seller.DeliveryRadiusKm,
		}

		sellerInfo = gin.H{
			"id":      seller.ID,
			"name":    seller.User.Email,
			"address": "Almaty",
		}
	} else {
		// Default empty delivery/seller if not found
		delivery = gin.H{
			"pickupEnabled":       false,
			"pickupTime":          nil,
			"pickupAddress":       nil,
			"freeDeliveryEnabled": false,
			"freeDeliveryText":    nil,
			"intercityEnabled":    false,
			"deliveryCenterLat":   nil,
			"deliveryCenterLng":   nil,
			"deliveryRadiusKm":    nil,
		}
		sellerInfo = gin.H{
			"id":      0,
			"name":    "Unknown",
			"address": "Unknown",
		}
	}

	appLinkBase := config.GetEnv("APP_LINK_BASE_URL", "https://ozmade-applink.vercel.app")

	response := ProductResponse{
		Product:   product,
		Delivery:  delivery,
		Seller:    sellerInfo,
		ShareLink: fmt.Sprintf("%s/products/%d", appLinkBase, product.ID),
	}

	c.JSON(http.StatusOK, response)
}

func ViewProduct(c *gin.Context) {
	id := c.Param("id")
	productID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	err = productservice.NewDefaultService().IncrementView(c.Request.Context(), uint(productID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update product view"})
		return
	}

	c.Status(http.StatusOK)
}

func GetTrendingProducts(c *gin.Context) {
	products, err := productservice.NewDefaultService().GetTrendingProducts(c.Request.Context(), 20)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch trending products"})
		return
	}

	c.JSON(http.StatusOK, buildProductResponses(products))
}

func buildProductResponses(products []models.Product) []ProductResponse {
	if len(products) == 0 {
		return []ProductResponse{}
	}

	var sellerIDs []uint
	for _, product := range products {
		sellerIDs = append(sellerIDs, product.SellerID)
	}

	sellerMap := make(map[uint]models.Seller)
	if len(sellerIDs) > 0 {
		var sellers []models.Seller
		database.DB.Preload("User").Where("id IN ?", sellerIDs).Find(&sellers)
		for _, seller := range sellers {
			sellerMap[seller.ID] = seller
		}
	}

	appLinkBase := config.GetEnv("APP_LINK_BASE_URL", "https://ozmade-applink.vercel.app")
	response := make([]ProductResponse, 0, len(products))

	for i := range products {
		url, _ := services.GenerateSignedURL(products[i].ImageName)
		products[i].ImageName = url

		for j, imgName := range products[i].Images {
			gURL, _ := services.GenerateSignedURL(imgName)
			products[i].Images[j] = gURL
		}

		seller, exists := sellerMap[products[i].SellerID]
		delivery := gin.H{}
		sellerInfo := gin.H{}

		if exists {
			products[i].SellerName = seller.User.Email
			delivery = gin.H{
				"pickupEnabled":       seller.PickupEnabled,
				"pickupTime":          seller.PickupTime,
				"pickupAddress":       seller.PickupAddress,
				"freeDeliveryEnabled": seller.FreeDeliveryEnabled,
				"freeDeliveryText":    "Citywide",
				"intercityEnabled":    seller.IntercityEnabled,
				"deliveryCenterLat":   seller.DeliveryCenterLat,
				"deliveryCenterLng":   seller.DeliveryCenterLng,
				"deliveryRadiusKm":    seller.DeliveryRadiusKm,
			}
			sellerInfo = gin.H{
				"id":      seller.ID,
				"name":    seller.User.Email,
				"address": "Almaty",
			}
		} else {
			delivery = gin.H{
				"pickupEnabled":       false,
				"pickupTime":          nil,
				"pickupAddress":       nil,
				"freeDeliveryEnabled": false,
				"freeDeliveryText":    nil,
				"intercityEnabled":    false,
				"deliveryCenterLat":   nil,
				"deliveryCenterLng":   nil,
				"deliveryRadiusKm":    nil,
			}
			sellerInfo = gin.H{
				"id":      0,
				"name":    "Unknown",
				"address": "Unknown",
			}
		}

		response = append(response, ProductResponse{
			Product:   products[i],
			Delivery:  delivery,
			Seller:    sellerInfo,
			ShareLink: fmt.Sprintf("%s/products/%d", appLinkBase, products[i].ID),
		})
	}

	return response
}
