package services

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"cloud.google.com/go/storage"
)

var GCS *GCSService

type GCSService struct {
	BucketName string
	Client     *storage.Client
	Creds      *ServiceAccountKey
}

type ServiceAccountKey struct {
	ClientEmail string `json:"client_email"`
	PrivateKey  string `json:"private_key"`
}

func InitGCSService(bucketName string, client *storage.Client) {
	if bucketName == "" {
		log.Println("WARNING: GCS_BUCKET_NAME is empty. Image uploads will fail.")
	} else {
		log.Printf("GCS Service initialized with bucket: %s\n", bucketName)
	}

	gcs := &GCSService{
		BucketName: bucketName,
		Client:     client,
	}

	// Load credentials for V4 signing if file exists
	credsPath := os.Getenv("FIREBASE_CREDENTIALS")
	if credsPath != "" {
		data, err := os.ReadFile(credsPath)
		if err == nil {
			var key ServiceAccountKey
			if err := json.Unmarshal(data, &key); err == nil {
				gcs.Creds = &key
				log.Println("GCS Service loaded signing credentials from service account")
			} else {
				log.Printf("Error unmarshaling service account key: %v\n", err)
			}
		} else {
			log.Printf("Error reading credentials file: %v\n", err)
		}
	}

	GCS = gcs
}

func (s *GCSService) GenerateSignedURL(objectName string, method string, expiry time.Duration, contentType string) (string, error) {
	if s.BucketName == "" {
		return "", fmt.Errorf("GCS_BUCKET_NAME is not configured")
	}

	// Important: To use storage.SignedURL (package-level), we MUST provide
	// GoogleAccessID and PrivateKey in the options.
	if s.Creds == nil || s.Creds.ClientEmail == "" || s.Creds.PrivateKey == "" {
		return "", fmt.Errorf("signing credentials are not properly configured")
	}

	opts := &storage.SignedURLOptions{
		Scheme:         storage.SigningSchemeV4,
		Method:         method,
		Expires:        time.Now().Add(expiry),
		ContentType:    contentType,
		GoogleAccessID: s.Creds.ClientEmail,
		PrivateKey:     []byte(s.Creds.PrivateKey),
	}

	u, err := storage.SignedURL(s.BucketName, objectName, opts)
	if err != nil {
		log.Printf("Failed to generate signed URL: %v\n", err)
		return "", err
	}
	return u, nil
}
