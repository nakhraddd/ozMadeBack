package handlers

import (
	"math"
	"math/rand"
	"net/http"
	"ozMadeBack/internal/database"
	"ozMadeBack/internal/models"
	"ozMadeBack/internal/services"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type OrderDto struct {
	ID                  uint      `json:"ID"`
	Status              string    `json:"Status"`
	CreatedAt           time.Time `json:"CreatedAt"`
	ProductID           uint      `json:"ProductID"`
	ProductTitle        string    `json:"ProductTitle"`
	ProductImageUrl     string    `json:"ProductImageUrl"`
	Price               float64   `json:"Price"`
	Quantity            int       `json:"Quantity"`
	TotalCost           float64   `json:"TotalCost"`
	SellerID            uint      `json:"SellerID"`
	SellerName          string    `json:"SellerName"`
	DeliveryType        string    `json:"DeliveryType"`
	PickupAddress       *string   `json:"PickupAddress"`
	PickupTime          *string   `json:"PickupTime"`
	ZoneCenterLat       *float64  `json:"ZoneCenterLat"`
	ZoneCenterLng       *float64  `json:"ZoneCenterLng"`
	ZoneRadiusKm        *float64  `json:"ZoneRadiusKm"`
	ZoneCenterAddress   *string   `json:"ZoneCenterAddress"`
	ShippingAddressText *string   `json:"ShippingAddressText"`
	ShippingLat         *float64  `json:"ShippingLat"`
	ShippingLng         *float64  `json:"ShippingLng"`
	ShippingComment     *string   `json:"ShippingComment"`
	ConfirmCode         *string   `json:"ConfirmCode"`
}

type CheckoutOptionDto struct {
	Code           string   `json:"code"`
	Title          string   `json:"title"`
	Enabled        bool     `json:"enabled"`
	RequiresFields []string `json:"requires_fields"`
	Description    string   `json:"description"`
}

func CreateOrder(c *gin.Context) {
	userID := c.GetUint("userID")

	var input struct {
		ProductID           uint     `json:"product_id"`
		Quantity            int      `json:"quantity"`
		DeliveryType        string   `json:"delivery_type"`
		ShippingAddressText *string  `json:"shipping_address_text"`
		ShippingLat         *float64 `json:"shipping_lat"`
		ShippingLng         *float64 `json:"shipping_lng"`
		ShippingComment     *string  `json:"shipping_comment"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Quantity < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Quantity must be at least 1"})
		return
	}

	// 2. Find product
	var product models.Product
	if err := database.DB.First(&product, input.ProductID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	// 3. Identify seller and fetch delivery settings
	var seller models.Seller
	if err := database.DB.Preload("User").First(&seller, product.SellerID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Seller not found"})
		return
	}

	if seller.UserID == userID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "You cannot order your own product"})
		return
	}

	// 7. Verify delivery type
	validDelivery := false
	switch input.DeliveryType {
	case models.DeliveryTypePickup:
		if seller.PickupEnabled {
			validDelivery = true
		}
	case models.DeliveryTypeMyDelivery:
		if seller.FreeDeliveryEnabled {
			validDelivery = true
			if input.ShippingAddressText == nil || *input.ShippingAddressText == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "shipping_address_text is required for MY_DELIVERY"})
				return
			}
			if !hasValidCoordinates(input.ShippingLat, input.ShippingLng) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "shipping_lat and shipping_lng are required for MY_DELIVERY"})
				return
			}
			if !hasValidCoordinates(&seller.DeliveryCenterLat, &seller.DeliveryCenterLng) || seller.DeliveryRadiusKm <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Seller delivery settings are incomplete"})
				return
			}

			distanceKm := calculateDistanceKm(
				seller.DeliveryCenterLat,
				seller.DeliveryCenterLng,
				*input.ShippingLat,
				*input.ShippingLng,
			)
			if distanceKm > seller.DeliveryRadiusKm {
				c.JSON(http.StatusBadRequest, gin.H{
					"error":                "Delivery is not available for this address",
					"delivery_radius_km":   seller.DeliveryRadiusKm,
					"distance_to_buyer_km": math.Round(distanceKm*100) / 100,
				})
				return
			}
		}
	case models.DeliveryTypeIntercity:
		if seller.IntercityEnabled {
			validDelivery = true
			if input.ShippingAddressText == nil || *input.ShippingAddressText == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "shipping_address_text is required for INTERCITY"})
				return
			}
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid delivery_type"})
		return
	}

	if !validDelivery {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Selected delivery type is not available for this seller"})
		return
	}

	// 9. Calculate costs
	totalCost := product.Cost * float64(input.Quantity)

	// 10. Create order
	// Generate a handoff code for manual completion flows.
	confirmCode := ""
	if input.DeliveryType == models.DeliveryTypePickup || input.DeliveryType == models.DeliveryTypeMyDelivery {
		confirmCode = strconv.Itoa(1000 + rand.Intn(9000)) // Simple 4 digit code
	}

	order := models.Order{
		UserID:              userID,
		ProductID:           product.ID,
		Quantity:            input.Quantity,
		TotalCost:           totalCost,
		Status:              models.StatusPendingSeller,
		CreatedAt:           time.Now(),
		DeliveryType:        input.DeliveryType,
		ShippingAddressText: input.ShippingAddressText,
		ShippingLat:         input.ShippingLat,
		ShippingLng:         input.ShippingLng,
		ShippingComment:     input.ShippingComment,
		ConfirmCode:         confirmCode,
	}

	if err := database.DB.Create(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create order"})
		return
	}

	// Construct DTO
	dto := mapOrderToDto(order, product, seller)
	c.JSON(http.StatusCreated, dto)
}

func GetCheckoutOptions(c *gin.Context) {
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

	var seller models.Seller
	if err := database.DB.Preload("User").First(&seller, product.SellerID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Seller not found"})
		return
	}

	options := []CheckoutOptionDto{
		{
			Code:           models.DeliveryTypePickup,
			Title:          "Pickup",
			Enabled:        seller.PickupEnabled,
			RequiresFields: []string{},
			Description:    buildPickupDescription(seller),
		},
		{
			Code:           models.DeliveryTypeMyDelivery,
			Title:          "Seller delivery",
			Enabled:        seller.FreeDeliveryEnabled,
			RequiresFields: []string{"shipping_address_text", "shipping_lat", "shipping_lng"},
			Description:    buildSellerDeliveryDescription(seller),
		},
		{
			Code:           models.DeliveryTypeIntercity,
			Title:          "Intercity delivery",
			Enabled:        seller.IntercityEnabled,
			RequiresFields: []string{"shipping_address_text"},
			Description:    "Requires shipping address",
		},
	}

	enabledOptions := make([]CheckoutOptionDto, 0, len(options))
	for _, option := range options {
		if option.Enabled {
			enabledOptions = append(enabledOptions, option)
		}
	}

	userAddress := ""
	if userID := c.GetUint("userID"); userID != 0 {
		var user models.User
		if err := database.DB.First(&user, userID).Error; err == nil {
			userAddress = user.Address
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"product_id":          product.ID,
		"product_title":       product.Title,
		"product_price":       product.Cost,
		"seller_id":           seller.ID,
		"seller_name":         resolveSellerDisplayName(seller),
		"buyer_saved_address": userAddress,
		"delivery_options":    enabledOptions,
		"delivery_summary":    serializeDeliverySettings(seller),
	})
}

func GetBuyerOrders(c *gin.Context) {
	userID := c.GetUint("userID")

	var orders []models.Order
	if err := database.DB.Where("user_id = ?", userID).Order("created_at desc").Find(&orders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch orders"})
		return
	}

	var dtos []OrderDto
	for _, order := range orders {
		var product models.Product
		var seller models.Seller

		// Error handling ignored for brevity in loop, but ideally should be handled
		database.DB.First(&product, order.ProductID)
		database.DB.Preload("User").First(&seller, product.SellerID)

		dtos = append(dtos, mapOrderToDto(order, product, seller))
	}

	c.JSON(http.StatusOK, dtos)
}

func CancelOrderBuyer(c *gin.Context) {
	orderID := c.Param("id")
	userID := c.GetUint("userID")

	var order models.Order
	if err := database.DB.Where("id = ? AND user_id = ?", orderID, userID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found or unauthorized"})
		return
	}

	if order.Status != models.StatusPendingSeller {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order cannot be cancelled in current status"})
		return
	}

	if err := database.DB.Model(&order).Update("status", models.StatusCancelledByBuyer).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel order"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Order cancelled"})
}

func BuyerReceived(c *gin.Context) {
	orderID := c.Param("id")
	userID := c.GetUint("userID")

	var order models.Order
	if err := database.DB.Where("id = ? AND user_id = ?", orderID, userID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found or unauthorized"})
		return
	}

	if order.DeliveryType != models.DeliveryTypeIntercity {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Received action only applicable for INTERCITY orders"})
		return
	}

	if order.Status != models.StatusReadyOrShipped {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order is not in shipped state"})
		return
	}

	if err := database.DB.Model(&order).Update("status", models.StatusCompleted).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update order status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Order marked as received"})
}

func usesManualCompletion(deliveryType string) bool {
	return deliveryType == models.DeliveryTypePickup || deliveryType == models.DeliveryTypeMyDelivery
}

func shouldExposeConfirmCode(order models.Order) bool {
	if !usesManualCompletion(order.DeliveryType) || order.ConfirmCode == "" {
		return false
	}

	switch order.Status {
	case models.StatusPendingSeller, models.StatusConfirmed, models.StatusReadyOrShipped:
		return true
	default:
		return false
	}
}

func hasValidCoordinates(lat, lng *float64) bool {
	return lat != nil && lng != nil
}

func calculateDistanceKm(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadiusKm = 6371.0

	toRadians := func(value float64) float64 {
		return value * math.Pi / 180
	}

	lat1Rad := toRadians(lat1)
	lat2Rad := toRadians(lat2)
	deltaLat := toRadians(lat2 - lat1)
	deltaLng := toRadians(lng2 - lng1)

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(deltaLng/2)*math.Sin(deltaLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusKm * c
}

func mapOrderToDto(order models.Order, product models.Product, seller models.Seller) OrderDto {
	imageUrl, _ := services.GenerateSignedURL(product.ImageName)

	dto := OrderDto{
		ID:                  order.ID,
		Status:              order.Status,
		CreatedAt:           order.CreatedAt,
		ProductID:           product.ID,
		ProductTitle:        product.Title,
		ProductImageUrl:     imageUrl,
		Price:               product.Cost,
		Quantity:            order.Quantity,
		TotalCost:           order.TotalCost,
		SellerID:            seller.ID,
		SellerName:          seller.User.Email, // Or seller name field
		DeliveryType:        order.DeliveryType,
		ShippingAddressText: order.ShippingAddressText,
		ShippingLat:         order.ShippingLat,
		ShippingLng:         order.ShippingLng,
		ShippingComment:     order.ShippingComment,
	}

	if shouldExposeConfirmCode(order) {
		dto.ConfirmCode = &order.ConfirmCode
	}

	// Fill delivery details based on type
	if order.DeliveryType == models.DeliveryTypePickup {
		dto.PickupAddress = &seller.PickupAddress
		dto.PickupTime = &seller.PickupTime
	} else if order.DeliveryType == models.DeliveryTypeMyDelivery {
		dto.ZoneCenterLat = &seller.DeliveryCenterLat
		dto.ZoneCenterLng = &seller.DeliveryCenterLng
		dto.ZoneRadiusKm = &seller.DeliveryRadiusKm
		dto.ZoneCenterAddress = &seller.DeliveryCenterAddress
	}

	return dto
}

func buildPickupDescription(seller models.Seller) string {
	if !seller.PickupEnabled {
		return ""
	}

	switch {
	case seller.PickupAddress != "" && seller.PickupTime != "":
		return seller.PickupAddress + ", " + seller.PickupTime
	case seller.PickupAddress != "":
		return seller.PickupAddress
	case seller.PickupTime != "":
		return seller.PickupTime
	default:
		return "Pickup available"
	}
}

func buildSellerDeliveryDescription(seller models.Seller) string {
	if !seller.FreeDeliveryEnabled {
		return ""
	}

	switch {
	case seller.DeliveryCenterAddress != "" && seller.DeliveryRadiusKm > 0:
		return seller.DeliveryCenterAddress + ", within " + strconv.FormatFloat(seller.DeliveryRadiusKm, 'f', -1, 64) + " km"
	case seller.DeliveryRadiusKm > 0:
		return "Within " + strconv.FormatFloat(seller.DeliveryRadiusKm, 'f', -1, 64) + " km"
	case seller.DeliveryCenterAddress != "":
		return seller.DeliveryCenterAddress
	default:
		return "Seller delivery available"
	}
}

func resolveSellerDisplayName(seller models.Seller) string {
	if seller.User.Name != "" {
		return seller.User.Name
	}
	if seller.User.Email != "" {
		return seller.User.Email
	}
	if seller.User.PhoneNumber != "" {
		return seller.User.PhoneNumber
	}
	return "Unknown"
}
