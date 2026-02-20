package services

import (
	"cloud.google.com/go/storage"
	"context"
	"fmt"
	"ozMadeBack/config"
	"time"
)

func GenerateSignedURL(objectName string) (string, error) {
	bucketName := config.GetEnv("GCS_BUCKET_NAME")
	opts := &storage.SignedURLOptions{
		Scheme:  storage.SigningSchemeV4,
		Method:  "GET",
		Expires: time.Now().Add(15 * time.Minute),
	}

	client, err := storage.NewClient(context.Background())
	if err != nil {
		return "", err
	}
	defer client.Close()

	u, err := client.Bucket(bucketName).SignedURL(objectName, opts)
	if err != nil {
		return "", fmt.Errorf("failed to sign URL: %v", err)
	}
	return u, nil
}
