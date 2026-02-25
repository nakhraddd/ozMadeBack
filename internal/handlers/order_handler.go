package handlers

import (
	"net/http"
	"ozMadeBack/internal/database"
	"ozMadeBack/internal/models"

	"github.com/gin-gonic/gin"
)

func CreateOrder(c *gin.Context) {
	userID := c.GetUint("userID")

	var input struct {
		ProductID uint    `json:"product_id"`
		Quantity  int     `json:"quantity"`
		TotalCost float64 `json:"total_cost"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	order := models.Order{
		UserID:    userID,
		ProductID: input.ProductID,
		Quantity:  input.Quantity,
		TotalCost: input.TotalCost,
		Status:    "pending",
	}

	if err := database.DB.Create(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create order"})
		return
	}

	c.JSON(http.StatusCreated, order)
}

func CancelOrderBuyer(c *gin.Context) {
	orderID := c.Param("id")
	userID := c.GetUint("userID")

	var order models.Order
	if err := database.DB.Where("id = ? AND user_id = ?", orderID, userID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found or unauthorized"})
		return
	}

	if err := database.DB.Model(&order).Update("status", "cancelled").Error; err != nil {
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

	if err := database.DB.Model(&order).Update("status", "received").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update order status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Order marked as received"})
}
