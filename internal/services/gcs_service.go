package services

import (
	"time"

	"cloud.google.com/go/storage"
)

var GCS *GCSService

type GCSService struct {
	BucketName string
	Client     *storage.Client
}

func InitGCSService(bucketName string, client *storage.Client) {
	GCS = &GCSService{
		BucketName: bucketName,
		Client:     client,
	}
}

func NewGCSService(bucketName string, client *storage.Client) *GCSService {
	return &GCSService{
		BucketName: bucketName,
		Client:     client,
	}
}

func (s *GCSService) GenerateSignedURL(objectName string, method string, expiry time.Duration, contentType string) (string, error) {
	opts := &storage.SignedURLOptions{
		Scheme:      storage.SigningSchemeV4,
		Method:      method,
		Expires:     time.Now().Add(expiry),
		ContentType: contentType,
	}

	u, err := s.Client.Bucket(s.BucketName).SignedURL(objectName, opts)
	if err != nil {
		return "", err
	}
	return u, nil
}
