package handlers

import (
	"math"
	"math/rand"
	"net/http"
	"ozMadeBack/internal/database"
	"ozMadeBack/internal/dto"
	"ozMadeBack/internal/models"
	productservice "ozMadeBack/internal/service/product"
	"ozMadeBack/internal/services"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func CreateOrder(c *gin.Context) {
	userID := c.GetUint("userID")

	var input dto.CreateOrderInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
		return
	}

	var buyer models.User
	if err := database.DB.First(&buyer, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "User not found"})
		return
	}

	var product models.Product
	if err := database.DB.First(&product, input.ProductID).Error; err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "Product not found"})
		return
	}

	var seller models.Seller
	if err := database.DB.Preload("User").First(&seller, product.SellerID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "Seller not found"})
		return
	}

	if seller.UserID == userID {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "You cannot order your own product"})
		return
	}

	finalShippingAddressText := input.ShippingAddressText
	finalShippingLat := input.ShippingLat
	finalShippingLng := input.ShippingLng

	if finalShippingAddressText == nil || *finalShippingAddressText == "" {
		if buyer.Address != "" {
			buyerAddress := buyer.Address
			finalShippingAddressText = &buyerAddress
		}
	}
	if finalShippingLat == nil && buyer.AddressLat != nil {
		finalShippingLat = buyer.AddressLat
	}
	if finalShippingLng == nil && buyer.AddressLng != nil {
		finalShippingLng = buyer.AddressLng
	}

	validDelivery := false
	switch input.DeliveryType {
	case models.DeliveryTypePickup:
		if seller.PickupEnabled {
			validDelivery = true
		}
	case models.DeliveryTypeMyDelivery:
		if seller.FreeDeliveryEnabled {
			validDelivery = true
			if finalShippingAddressText == nil || *finalShippingAddressText == "" {
				c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "shipping_address_text is required for MY_DELIVERY"})
				return
			}
			if !hasValidCoordinates(finalShippingLat, finalShippingLng) {
				c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "shipping_lat and shipping_lng are required for MY_DELIVERY"})
				return
			}
			if !hasValidCoordinates(&seller.DeliveryCenterLat, &seller.DeliveryCenterLng) || seller.DeliveryRadiusKm <= 0 {
				c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "Seller delivery settings are incomplete"})
				return
			}

			distanceKm := calculateDistanceKm(
				seller.DeliveryCenterLat,
				seller.DeliveryCenterLng,
				*finalShippingLat,
				*finalShippingLng,
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
			if finalShippingAddressText == nil || *finalShippingAddressText == "" {
				c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "shipping_address_text is required for INTERCITY"})
				return
			}
		}
	default:
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "Invalid delivery_type"})
		return
	}

	if !validDelivery {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "Selected delivery type is not available for this seller"})
		return
	}

	totalCost := product.Cost * float64(input.Quantity)

	confirmCode := ""
	if input.DeliveryType == models.DeliveryTypePickup || input.DeliveryType == models.DeliveryTypeMyDelivery {
		confirmCode = strconv.Itoa(1000 + rand.Intn(9000))
	}

	order := models.Order{
		UserID:              userID,
		ProductID:           product.ID,
		Quantity:            input.Quantity,
		TotalCost:           totalCost,
		Status:              models.StatusPendingSeller,
		CreatedAt:           time.Now(),
		DeliveryType:        input.DeliveryType,
		ShippingAddressText: finalShippingAddressText,
		ShippingLat:         finalShippingLat,
		ShippingLng:         finalShippingLng,
		ShippingComment:     input.ShippingComment,
		ConfirmCode:         confirmCode,
	}

	if err := database.DB.Create(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "Failed to create order"})
		return
	}

	_ = productservice.NewDefaultService().IncrementOrderCount(c.Request.Context(), product.ID)

	orderID := order.ID
	_ = services.CreateNotification(userID, "Order created", "Your order has been created and is waiting for seller confirmation.", "buyer_order_created", &orderID, nil)
	_ = services.CreateNotification(seller.UserID, "New order received", "A buyer has placed a new order for your product.", "seller_new_order", &orderID, nil)

	c.JSON(http.StatusCreated, mapOrderToDto(order, product, seller))
}

func GetCheckoutOptions(c *gin.Context) {
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

	var seller models.Seller
	if err := database.DB.Preload("User").First(&seller, product.SellerID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "Seller not found"})
		return
	}

	options := []dto.CheckoutOptionDto{
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

	enabledOptions := make([]dto.CheckoutOptionDto, 0)
	for _, option := range options {
		if option.Enabled {
			enabledOptions = append(enabledOptions, option)
		}
	}

	userAddress := ""
	var userAddressLat *float64
	var userAddressLng *float64
	if userID := c.GetUint("userID"); userID != 0 {
		var user models.User
		if err := database.DB.First(&user, userID).Error; err == nil {
			userAddress = user.Address
			userAddressLat = user.AddressLat
			userAddressLng = user.AddressLng
		}
	}

	productImageUrl, _ := services.GenerateSignedURL(product.ImageName)

	c.JSON(http.StatusOK, dto.ProductCheckoutOptionsResponse{
		ProductID:            product.ID,
		ProductTitle:         product.Title,
		ProductPrice:         product.Cost,
		ProductImageUrl:      productImageUrl,
		SellerID:             seller.ID,
		SellerName:           resolveSellerDisplayName(seller),
		BuyerSavedAddress:    userAddress,
		BuyerSavedAddressLat: userAddressLat,
		BuyerSavedAddressLng: userAddressLng,
		DeliveryOptions:      enabledOptions,
		DeliverySummary:      serializeDeliverySettings(seller),
	})
}

func GetBuyerOrders(c *gin.Context) {
	userID := c.GetUint("userID")

	var orders []models.Order
	if err := database.DB.Where("user_id = ?", userID).Order("created_at desc").Find(&orders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "Failed to fetch orders"})
		return
	}

	dtos := make([]dto.OrderDto, 0)
	for _, order := range orders {
		var product models.Product
		var seller models.Seller

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
		c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "Order not found or unauthorized"})
		return
	}

	if order.Status != models.StatusPendingSeller {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "Order cannot be cancelled in current status"})
		return
	}

	if err := database.DB.Model(&order).Update("status", models.StatusCancelledByBuyer).Error; err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "Failed to cancel order"})
		return
	}

	if sellerUserID, err := findSellerUserIDByProductID(order.ProductID); err == nil {
		orderRecordID := order.ID
		_ = services.CreateNotification(sellerUserID, "Order cancelled", "The buyer cancelled the order.", "seller_order_cancelled", &orderRecordID, nil)
	}

	c.JSON(http.StatusOK, dto.MessageResponse{Message: "Order cancelled"})
}

func BuyerReceived(c *gin.Context) {
	orderID := c.Param("id")
	userID := c.GetUint("userID")

	var order models.Order
	if err := database.DB.Where("id = ? AND user_id = ?", orderID, userID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "Order not found or unauthorized"})
		return
	}

	if order.DeliveryType != models.DeliveryTypeIntercity {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "Received action only applicable for INTERCITY orders"})
		return
	}

	if order.Status != models.StatusReadyOrShipped {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "Order is not in shipped state"})
		return
	}

	if err := database.DB.Model(&order).Update("status", models.StatusCompleted).Error; err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "Failed to update order status"})
		return
	}

	if sellerUserID, err := findSellerUserIDByProductID(order.ProductID); err == nil {
		orderRecordID := order.ID
		_ = services.CreateNotification(sellerUserID, "Order received", "The buyer confirmed that the order was received.", "seller_order_completed", &orderRecordID, nil)
	}

	c.JSON(http.StatusOK, dto.MessageResponse{Message: "Order marked as received"})
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

func mapOrderToDto(order models.Order, product models.Product, seller models.Seller) dto.OrderDto {
	imageUrl, _ := services.GenerateSignedURL(product.ImageName)

	res := dto.OrderDto{
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
		SellerName:          resolveSellerDisplayName(seller),
		DeliveryType:        order.DeliveryType,
		ShippingAddressText: order.ShippingAddressText,
		ShippingLat:         order.ShippingLat,
		ShippingLng:         order.ShippingLng,
		ShippingComment:     order.ShippingComment,
	}

	if shouldExposeConfirmCode(order) {
		res.ConfirmCode = &order.ConfirmCode
	}

	if order.DeliveryType == models.DeliveryTypePickup {
		res.PickupAddress = &seller.PickupAddress
		res.PickupTime = &seller.PickupTime
	} else if order.DeliveryType == models.DeliveryTypeMyDelivery {
		res.ZoneCenterLat = &seller.DeliveryCenterLat
		res.ZoneCenterLng = &seller.DeliveryCenterLng
		res.ZoneRadiusKm = &seller.DeliveryRadiusKm
		res.ZoneCenterAddress = &seller.DeliveryCenterAddress
	}

	return res
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
	if seller.DisplayName != "" {
		return seller.DisplayName
	}
	if seller.User.Name != "" {
		return seller.User.Name
	}
	if seller.User.PhoneNumber != "" {
		return seller.User.PhoneNumber
	}
	return "Unknown"
}

func findSellerUserIDByProductID(productID uint) (uint, error) {
	var product models.Product
	if err := database.DB.Select("seller_id").First(&product, productID).Error; err != nil {
		return 0, err
	}

	var seller models.Seller
	if err := database.DB.Select("user_id").First(&seller, product.SellerID).Error; err != nil {
		return 0, err
	}

	return seller.UserID, nil
}
