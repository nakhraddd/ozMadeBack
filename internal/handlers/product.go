package handlers

import (
	"net/http"
	"ozMadeBack/internal/database"
	"ozMadeBack/internal/models"
	"ozMadeBack/internal/services"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ProductResponse struct {
	models.Product
	Delivery gin.H `json:"delivery"`
	Seller   gin.H `json:"seller"`
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
			Product:  products[i],
			Delivery: delivery,
			Seller:   sellerInfo,
		})
	}

	c.JSON(http.StatusOK, response)
}

func GetProduct(c *gin.Context) {
	id := c.Param("id")
	var product models.Product
	if err := database.DB.Preload("Comments").First(&product, id).Error; err != nil {
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

	response := ProductResponse{
		Product:  product,
		Delivery: delivery,
		Seller:   sellerInfo,
	}

	c.JSON(http.StatusOK, response)
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
	var response []ProductResponse

	if len(ids) > 0 {
		database.DB.Where("id IN ?", ids).Find(&products)

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
				productSellerName := seller.User.Email // Keeping existing logic
				products[i].SellerName = productSellerName

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
				Product:  products[i],
				Delivery: delivery,
				Seller:   sellerInfo,
			})
		}
	} else {
		response = []ProductResponse{}
	}

	c.JSON(http.StatusOK, response)
}
