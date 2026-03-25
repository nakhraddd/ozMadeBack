package realtime

import (
	"context"
	"log"

	"ozMadeBack/internal/auth"

	"firebase.google.com/go/v4/messaging"
)

// SendFCMNotification sends a push notification to a specific device.
func SendFCMNotification(token string, title string, body string, data map[string]string) error {
	client := auth.GetFCMClient()
	if client == nil {
		log.Println("FCM client not initialized")
		return nil // Or return an error
	}

	message := &messaging.Message{
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Token: token,
		Data:  data,
		Android: &messaging.AndroidConfig{
			Priority: "high",
		},
	}

	response, err := client.Send(context.Background(), message)
	if err != nil {
		log.Printf("Error sending FCM message: %v\n", err)
		return err
	}

	log.Printf("Successfully sent FCM message: %s\n", response)
	return nil
}
