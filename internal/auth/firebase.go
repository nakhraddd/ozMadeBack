package auth

import (
	"context"
	"firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/option"
	"log"
	"net/http"
	"os"
	"ozMadeBack/config"
	"ozMadeBack/internal/database"
	"ozMadeBack/internal/models"
	"strings"
)

var Client *auth.Client

func InitFirebase() {
	serviceAccountPath := config.GetEnv("FIREBASE_CREDENTIALS")
	if serviceAccountPath == "" {
		serviceAccountPath = config.GetEnv("FIREBASE_SERVICE_ACCOUNT_JSON_PATH")
	}

	if serviceAccountPath == "" {
		log.Fatal("FIREBASE_CREDENTIALS environment variable is not set")
	}

	if _, err := os.Stat(serviceAccountPath); os.IsNotExist(err) {
		log.Fatalf("Firebase service account file not found at: %s", serviceAccountPath)
	}

	opt := option.WithCredentialsFile(serviceAccountPath)
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		log.Fatalf("error initializing app: %v\n", err)
	}
	Client, err = app.Auth(context.Background())
	if err != nil {
		log.Fatalf("error getting Auth client: %v\n", err)
	}
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header not provided"})
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		token, err := Client.VerifyIDToken(context.Background(), tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid ID token"})
			return
		}

		c.Set("firebaseToken", token)

		var user models.User
		if err := database.DB.Where("firebase_uid = ?", token.UID).First(&user).Error; err != nil {
			// If user not found, we might still want to proceed for SyncUser endpoint
			// But for other endpoints, we might want to block.
			// However, SyncUser is protected by this middleware.
			// If SyncUser is called, the user might not exist in DB yet.

			// Check if the request path is /auth/sync, if so, allow proceed without user_id
			if c.FullPath() == "/auth/sync" {
				c.Next()
				return
			}

			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "User not found"})
			return
		}

		c.Set("user_id", user.ID)
		c.Next()
	}
}
