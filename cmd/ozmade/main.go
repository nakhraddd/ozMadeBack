package main

import (
	"context"
	"log"
	"os"

	"ozMadeBack/config"
	"ozMadeBack/internal/auth"
	"ozMadeBack/internal/database"
	"ozMadeBack/internal/handlers"
	"ozMadeBack/internal/routes"
	"ozMadeBack/internal/services"
	"ozMadeBack/pkg/realtime"

	"cloud.google.com/go/storage"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/option"
)

func main() {
	config.LoadEnv()

	database.Connect(os.Getenv("DATABASE_URL"))
	database.Migrate()
	database.InitRedis()
	services.InitProductSearchService()

	auth.InitFirebase()

	// Start background worker
	go services.StartTrendingWorker()

	// Start WebSocket Hub
	hub := realtime.GetHub()
	go hub.Run()

	credsPath := os.Getenv("FIREBASE_CREDENTIALS")
	if credsPath == "" {
		log.Fatal("FIREBASE_CREDENTIALS environment variable is not set")
	}

	opt := option.WithCredentialsFile(credsPath)
	storageClient, err := storage.NewClient(context.Background(), opt)
	if err != nil {
		log.Fatalf("error creating storage client: %v\n", err)
	}

	bucketName := os.Getenv("GCS_BUCKET_NAME")
	if bucketName == "" {
		// FALLBACK: If GCS_BUCKET_NAME is not set, try to use the project ID from Firebase as a guess
		// but it's better to log an error.
		log.Println("CRITICAL ERROR: GCS_BUCKET_NAME environment variable is empty!")
	} else {
		log.Printf("Initializing GCS with bucket: %s\n", bucketName)
	}

	services.InitGCSService(bucketName, storageClient)
	services.BootstrapProductIndex(context.Background())
	sellerHandler := handlers.NewSellerHandler(services.GCS)

	r := gin.Default()

	// WebSocket route
	r.GET("/ws", realtime.HandleWebSocket)

	routes.SetupRoutes(r)
	routes.SellerRoutes(r, sellerHandler, auth.Client)

	r.Run("0.0.0.0:8080")
}
