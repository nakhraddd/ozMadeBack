package handlers

import (
	"net/http"
	"ozMadeBack/internal/database"
	"ozMadeBack/internal/models"

	"github.com/gin-gonic/gin"
)

func GetNotifications(c *gin.Context) {
	userID := c.GetUint("userID")

	var notifications []models.Notification
	if err := database.DB.Where("user_id = ?", userID).Order("created_at desc").Find(&notifications).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch notifications"})
		return
	}

	c.JSON(http.StatusOK, notifications)
}

func MarkNotificationRead(c *gin.Context) {
	userID := c.GetUint("userID")
	notificationID := c.Param("id")

	var notification models.Notification
	if err := database.DB.Where("id = ? AND user_id = ?", notificationID, userID).First(&notification).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Notification not found"})
		return
	}

	if err := database.DB.Model(&notification).Update("is_read", true).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update notification"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Notification marked as read"})
}

func MarkAllNotificationsRead(c *gin.Context) {
	userID := c.GetUint("userID")

	if err := database.DB.Model(&models.Notification{}).
		Where("user_id = ? AND is_read = false", userID).
		Update("is_read", true).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update notifications"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "All notifications marked as read"})
}
