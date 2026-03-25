package handlers

import (
	"errors"
	"math/rand"
	"net/http"
	"ozMadeBack/internal/database"
	"ozMadeBack/internal/models"
	"ozMadeBack/internal/services"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Constants for Order Statuses and Delivery Types
const (
	StatusPendingSeller     = "PENDING_SELLER"
	StatusConfirmed         = "CONFIRMED"
	StatusReadyOrShipped    = "READY_OR_SHIPPED"
	StatusCompleted         = "COMPLETED"
	StatusCancelledByBuyer  = "CANCELLED_BY_BUYER"
	StatusCancelledBySeller = "CANCELLED_BY_SELLER"
	StatusExpired           = "EXPIRED"

	DeliveryTypePickup     = "PICKUP"
	DeliveryTypeMyDelivery = "MY_DELIVERY"
	DeliveryTypeIntercity  = "INTERCITY"
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
	ShippingComment     *string   `json:"ShippingComment"`
	ConfirmCode         *string   `json:"ConfirmCode"`
}

func CreateOrder(c *gin.Context) {
	userID := c.GetUint("userID")

	var input struct {
		ProductID           uint    `json:"product_id"`
		Quantity            int     `json:"quantity"`
		DeliveryType        string  `json:"delivery_type"`
		ShippingAddressText *string `json:"shipping_address_text"`
		ShippingComment     *string `json:"shipping_comment"`
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

	// 7. Verify delivery type
	validDelivery := false
	switch input.DeliveryType {
	case DeliveryTypePickup:
		if seller.PickupEnabled {
			validDelivery = true
		}
	case DeliveryTypeMyDelivery: // Maps to FreeDeliveryEnabled based on context, or needs explicit mapping. Assuming FreeDeliveryEnabled covers local delivery by seller.
		if seller.FreeDeliveryEnabled {
			validDelivery = true
		}
	case DeliveryTypeIntercity:
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
	// Generate confirm code for Pickup/MyDelivery
	confirmCode := ""
	if input.DeliveryType == DeliveryTypePickup || input.DeliveryType == DeliveryTypeMyDelivery {
		confirmCode = strconv.Itoa(1000 + rand.Intn(9000)) // Simple 4 digit code
	}

	order := models.Order{
		UserID:              userID,
		ProductID:           product.ID,
		Quantity:            input.Quantity,
		TotalCost:           totalCost,
		Status:              StatusPendingSeller,
		CreatedAt:           time.Now(),
		DeliveryType:        input.DeliveryType,
		ShippingAddressText: input.ShippingAddressText,
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

	if order.Status != StatusPendingSeller {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order cannot be cancelled in current status"})
		return
	}

	if err := database.DB.Model(&order).Update("status", StatusCancelledByBuyer).Error; err != nil {
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

	if order.DeliveryType != DeliveryTypeIntercity {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Received action only applicable for INTERCITY orders"})
		return
	}

	if order.Status != StatusReadyOrShipped {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order is not in shipped state"})
		return
	}

	if err := database.DB.Model(&order).Update("status", StatusCompleted).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update order status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Order marked as received"})
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
		ShippingComment:     order.ShippingComment,
		// ConfirmCode:         &order.ConfirmCode, // Only expose if needed or logic dictates
	}

	// Conditionally expose confirm code to buyer if status is appropriate, usually buyer sees it to give to seller?
	// Spec says: "Seller completes the order via POST /seller/orders/{id}/complete. Request: { "code": "1234" }"
	// This implies Buyer has the code.
	if order.Status == StatusConfirmed || order.Status == StatusReadyOrShipped {
		dto.ConfirmCode = &order.ConfirmCode
	}

	// Fill delivery details based on type
	if order.DeliveryType == DeliveryTypePickup {
		dto.PickupAddress = &seller.PickupAddress
		dto.PickupTime = &seller.PickupTime
	} else if order.DeliveryType == DeliveryTypeMyDelivery {
		dto.ZoneCenterLat = &seller.DeliveryCenterLat
		dto.ZoneCenterLng = &seller.DeliveryCenterLng
		dto.ZoneRadiusKm = &seller.DeliveryRadiusKm
		dto.ZoneCenterAddress = &seller.DeliveryCenterAddress
	}

	return dto
}
