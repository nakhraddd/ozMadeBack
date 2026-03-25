package auth

import (
	"context"
	"log"
	"net/http"
	"os"
	"ozMadeBack/config"
	"ozMadeBack/internal/database"
	"ozMadeBack/internal/models"
	"strings"

	"firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"firebase.google.com/go/v4/messaging"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/option"
)

var (
	Client *auth.Client
	FCM    *messaging.Client
)

func InitFirebase() {
	serviceAccountPath := config.GetEnv("FIREBASE_CREDENTIALS", "$HOME/firebase_credentials.json")
	if serviceAccountPath == "" {
		serviceAccountPath = config.GetEnv("FIREBASE_SERVICE_ACCOUNT_JSON_PATH", "$HOME/firebase_credentials.json")
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

	FCM, err = app.Messaging(context.Background())
	if err != nil {
		log.Fatalf("error getting Messaging client: %v\n", err)
	}
}

func GetFCMClient() *messaging.Client {
	return FCM
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

		c.Set("userID", user.ID) // Using "userID" consistently as per handler usage
		c.Next()
	}
}
