package auth

import (
	"context"
	"firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/option"
	"log"
	"net/http"
	"ozMadeBack/config"
	"strings"
)

var Client *auth.Client

func InitFirebase() {
	opt := option.WithCredentialsFile(config.GetEnv("FIREBASE_SERVICE_ACCOUNT_JSON_PATH"))
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
		c.Next()
	}
}
