package middleware

import (
	"context"
	"net/http"
	"strings"

	"ozMadeBack/internal/database"
	"ozMadeBack/internal/models"

	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm" // Import gorm for ErrRecordNotFound
)

func AuthMiddleware(authClient *auth.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string
		authHeader := c.GetHeader("Authorization")

		if authHeader != "" {
			tokenString = strings.TrimPrefix(authHeader, "Bearer ")
		} else {
			// Try to get from cookie for browser access
			cookie, err := c.Cookie("token")
			if err == nil {
				tokenString = cookie
			}
		}

		if tokenString == "" {
			if strings.HasPrefix(c.Request.URL.Path, "/admin/ui") {
				c.Redirect(http.StatusFound, "/admin/login")
				c.Abort()
				return
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization token missing"})
			return
		}

		token, err := authClient.VerifyIDToken(context.Background(), tokenString)
		if err != nil {
			if strings.HasPrefix(c.Request.URL.Path, "/admin/ui") {
				c.Redirect(http.StatusFound, "/admin/login")
				c.Abort()
				return
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		var user models.User
		if err := database.DB.Where("firebase_uid = ?", token.UID).First(&user).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				// Special case for sync endpoint
				if c.FullPath() == "/auth/sync" {
					c.Set("firebaseToken", token)
					c.Next()
					return
				}
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "User not found in database. Please sync first."})
				return
			}
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
			return
		}

		c.Set("userID", user.ID)
		c.Set("user", user)
		c.Set("userRole", user.Role) // Set user role in context for AdminMiddleware
		c.Set("firebaseToken", token)
		c.Next()
	}
}

func SellerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// AuthMiddleware should have already run and set "user" and "userID"
		userVal, exists := c.Get("user")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			return
		}

		user, ok := userVal.(models.User)
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Invalid user object in context"})
			return
		}

		if !user.IsSeller {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Access denied: Seller profile required"})
			return
		}

		c.Next()
	}
}

// AdminMiddleware checks if the authenticated user has an 'admin' role
func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userRoleVal, exists := c.Get("userRole")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "User role not found in context"})
			return
		}

		userRole, ok := userRoleVal.(string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Invalid user role type in context"})
			return
		}

		if userRole != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Access denied: Admin role required"})
			return
		}
		c.Next()
	}
}
