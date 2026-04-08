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
			parts := strings.SplitN(u.Path, "/", 3)
			if len(parts) == 3 {
				objectName = parts[2] // keep everything after bucket name
			}
		}
	}

	// Ensure the product image has the "products/" prefix
	// (only for GET requests – upload URLs are generated separately)
	if !strings.HasPrefix(objectName, "products/") && !strings.HasPrefix(objectName, "seller_ids/") {
		objectName = "products/" + objectName
	}

	return GCS.GenerateSignedURL(objectName, "GET", 15*time.Minute, "")
}
