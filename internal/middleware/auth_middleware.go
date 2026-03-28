package middleware

import (
	"context"
	"net/http"
	"strings"

	"ozMadeBack/internal/database"
	"ozMadeBack/internal/models"

	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
)

func AuthMiddleware(authClient *auth.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header missing"})
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		token, err := authClient.VerifyIDToken(context.Background(), tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		var user models.User
		if err := database.DB.Where("firebase_uid = ?", token.UID).First(&user).Error; err != nil {
			// Special case for sync endpoint
			if c.FullPath() == "/auth/sync" {
				c.Set("firebaseToken", token)
				c.Next()
				return
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "User not found in database. Please sync first."})
			return
		}

		c.Set("userID", user.ID)
		c.Set("user", user)
		c.Set("firebaseToken", token)
		c.Next()
	}
}

func SellerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetUint("userID")
		if userID == 0 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			return
		}

		var user models.User
		if err := database.DB.First(&user, userID).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
			return
		}

		if !user.IsSeller {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Access denied: Seller profile required"})
			return
		}

		c.Next()
	}
}
