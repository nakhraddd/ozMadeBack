package routes

import (
	"ozMadeBack/internal/handlers"
	"ozMadeBack/internal/middleware"

	firebaseAuth "firebase.google.com/go/v4/auth" // Alias firebase auth package
	"github.com/gin-gonic/gin"
)

// SetupAdminRoutes initializes all admin-specific routes
func SetupAdminRoutes(r *gin.Engine, adminHandler *handlers.AdminHandler, authClient *firebaseAuth.Client) {
	adminGroup := r.Group("/admin")
	// Admin-specific authentication and authorization middleware
	adminGroup.Use(middleware.AuthMiddleware(authClient)) // Use actual Firebase auth client
	adminGroup.Use(middleware.AdminMiddleware())          // Check for admin role
	{
		// User Management Routes
		adminGroup.GET("/users", adminHandler.GetUsers)
		adminGroup.GET("/users/:id", adminHandler.GetUser)
		adminGroup.POST("/users", adminHandler.CreateUser)
		adminGroup.PUT("/users/:id", adminHandler.UpdateUser)
		adminGroup.DELETE("/users/:id", adminHandler.DeleteUser)

		// Product Review Routes
		adminGroup.GET("/products/pending-review", adminHandler.GetPendingReviewProducts)
		adminGroup.POST("/products/:id/approve", adminHandler.ApproveProduct)
		adminGroup.POST("/products/:id/reject", adminHandler.RejectProduct)

		// Seller Licenses Route
		adminGroup.GET("/sellers/:id/licenses", adminHandler.GetSellerLicenses) // New: Get seller licenses

		// Report Review Routes
		adminGroup.GET("/reports", adminHandler.GetReports)
		adminGroup.GET("/reports/:id", adminHandler.GetReport)
		adminGroup.POST("/reports/:id/resolve", adminHandler.ResolveReport)
		adminGroup.POST("/reports/:id/dismiss", adminHandler.DismissReport)
	}
}
