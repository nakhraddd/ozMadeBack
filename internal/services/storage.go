package services

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

func GenerateSignedURL(objectName string) (string, error) {
	if GCS == nil {
		return "", fmt.Errorf("GCS service not initialized")
	}

	// If objectName is empty, return empty (no URL)
	if objectName == "" {
		return "", nil
	}

	// If objectName is already a full signed URL, extract the object path
	if strings.HasPrefix(objectName, "https://storage.googleapis.com/") {
		// Parse the URL to get the path after the bucket name
		u, err := url.Parse(objectName)
		if err == nil {
			// Path is like "/oz-made/products/xxx.jpg"
			parts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
			if len(parts) >= 2 {
				// parts[0] is the bucket name, join the rest
				objectName = strings.Join(parts[1:], "/")
			}
		}
	}

	// Clean the path
	objectName = strings.TrimPrefix(objectName, "/")

	// Ensure the product image has the "products/" prefix if it doesn't have another directory prefix
	if !strings.HasPrefix(objectName, "products/") && !strings.HasPrefix(objectName, "seller_ids/") && !strings.HasPrefix(objectName, "chats/") && !strings.HasPrefix(objectName, "users/") {
		objectName = "products/" + objectName
	}

	return GCS.GenerateSignedURL(objectName, "GET", 15*time.Minute, "")
}

func GenerateSignedURLForChat(objectName string) (string, error) {
	if GCS == nil {
		return "", fmt.Errorf("GCS service not initialized")
	}

	if objectName == "" {
		return "", nil
	}

	// Clean the path
	objectName = strings.TrimPrefix(objectName, "/")

	return GCS.GenerateSignedURL(objectName, "GET", 24*time.Hour, "")
}

func GenerateSignedURLForUser(objectName string) (string, error) {
	if GCS == nil {
		return "", fmt.Errorf("GCS service not initialized")
	}

	if objectName == "" {
		return "", nil
	}

	// Clean the path
	objectName = strings.TrimPrefix(objectName, "/")

	return GCS.GenerateSignedURL(objectName, "GET", 24*time.Hour, "")
}
