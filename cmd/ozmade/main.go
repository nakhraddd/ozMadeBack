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

	"cloud.google.com/go/storage"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/option"
)

func main() {
	config.LoadEnv()

	database.Connect(os.Getenv("DATABASE_URL"))
	database.Migrate()
	database.InitRedis()

	auth.InitFirebase()

	// Start background worker
	go services.StartTrendingWorker()

	opt := option.WithCredentialsFile(os.Getenv("FIREBASE_CREDENTIALS"))
	storageClient, err := storage.NewClient(context.Background(), opt)
	if err != nil {
		log.Fatalf("error creating storage client: %v\n", err)
	}

	services.InitGCSService(os.Getenv("GCS_BUCKET_NAME"), storageClient)
	sellerHandler := handlers.NewSellerHandler(services.GCS)

	r := gin.Default()

	routes.SetupRoutes(r)
	routes.SellerRoutes(r, sellerHandler, auth.Client)

	r.Run("0.0.0.0:8080")
}
