package services

import (
	"log"
	"strconv"

	"ozMadeBack/internal/database"
	"ozMadeBack/internal/models"
	"ozMadeBack/pkg/realtime"
)

func CreateNotification(userID uint, title, body, notificationType string, orderID *uint, data map[string]string) error {
	notification := models.Notification{
		UserID:  userID,
		Title:   title,
		Body:    body,
		Type:    notificationType,
		OrderID: orderID,
		IsRead:  false,
	}

	if err := database.DB.Create(&notification).Error; err != nil {
		log.Printf("Error creating notification record: %v", err)
		return err
	}

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return nil
	}

	if user.FCMToken == "" {
		return nil
	}

	payload := map[string]string{
		"notification_id": strconv.FormatUint(uint64(notification.ID), 10),
		"type":            notificationType,
	}
	if orderID != nil {
		payload["order_id"] = strconv.FormatUint(uint64(*orderID), 10)
	}
	for key, value := range data {
		payload[key] = value
	}

	err := realtime.SendFCMNotification(user.FCMToken, title, body, payload)
	if err != nil {
		log.Printf("Error sending FCM notification to user %d: %v", userID, err)
	}

	return err
}
