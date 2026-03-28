package services

import (
	"fmt"
	"time"
)

func GenerateSignedURL(objectName string) (string, error) {
	if GCS == nil {
		return "", fmt.Errorf("GCS service not initialized")
	}
	return GCS.GenerateSignedURL(objectName, "GET", 15*time.Minute, "")
}
