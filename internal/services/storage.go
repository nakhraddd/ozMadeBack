package services

import (
	"fmt"
	"strings"
	"time"
)

func GenerateSignedURL(objectName string) (string, error) {
	if GCS == nil {
		return "", fmt.Errorf("GCS service not initialized")
	}

	// If objectName doesn't already start with 'products/', prepend it
	// to match the expected path: oz-made/products/image
	if objectName != "" && !strings.HasPrefix(objectName, "products/") {
		objectName = "products/" + objectName
	}

	return GCS.GenerateSignedURL(objectName, "GET", 15*time.Minute, "")
}
