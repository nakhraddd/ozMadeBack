package routes

import (
	"ozMadeBack/internal/handlers"
	"ozMadeBack/internal/middleware" // Import the new middleware package

	firebaseAuth "firebase.google.com/go/v4/auth" // Alias firebase auth package
	"github.com/gin-gonic/gin"
)

// SetupRoutes initializes all application routes
func SetupRoutes(r *gin.Engine, sellerHandler *handlers.SellerHandler, adminHandler *handlers.AdminHandler, orderHandler *handlers.OrderHandler, authClient *firebaseAuth.Client) {
	// Public routes
	r.GET("/categories", handlers.GetCategories)
	r.GET("/ads", handlers.GetAds)

	r.GET("/sellers/:id", sellerHandler.GetSellerProfile)
	r.GET("/sellers/:id/reviews", sellerHandler.GetSellerQuality) // seller quality metrics and reviews

	authRoutes := r.Group("/auth")
	authRoutes.Use(middleware.AuthMiddleware(authClient)) // Use the new middleware
	{
		authRoutes.POST("/sync", handlers.SyncUser)
	}

	productRoutes := r.Group("/products")
	{
		productRoutes.GET("/search", handlers.SearchProducts)
		productRoutes.GET("/trending", handlers.GetTrendingProducts)
		productRoutes.GET("/recommendations", middleware.AuthMiddleware(authClient), handlers.GetRecommendations)      // Use the new middleware
		productRoutes.GET("/:id/checkout-options", middleware.AuthMiddleware(authClient), handlers.GetCheckoutOptions) // Use the new middleware
		productRoutes.GET("", handlers.GetProducts)
		productRoutes.GET("/:id", handlers.GetProduct)
		productRoutes.GET("/:id/reviews", handlers.GetProductReviews)
		productRoutes.POST("/:id/view", handlers.ViewProduct)

		productRoutes.POST("/:id/comments", middleware.AuthMiddleware(authClient), handlers.PostComment) // Protected, use new middleware
		productRoutes.POST("/:id/report", middleware.AuthMiddleware(authClient), handlers.ReportProduct) // Protected, use new middleware
	}

	userRoutes := r.Group("/profile")
	userRoutes.Use(middleware.AuthMiddleware(authClient)) // Use the new middleware
	{
		userRoutes.GET("", handlers.GetProfile)
		userRoutes.PATCH("", handlers.UpdateProfile)
		userRoutes.PATCH("/fcm-token", handlers.UpdateFCMToken)
		userRoutes.POST("/favorites/:id", handlers.ToggleFavorite)
		userRoutes.GET("/favorites", handlers.GetFavorites)
		userRoutes.GET("/orders", handlers.GetBuyerOrders) // Use orderHandler
		userRoutes.GET("/upload-url", handlers.GetProfileUploadURL)
	}

	notificationRoutes := r.Group("/notifications")
	notificationRoutes.Use(middleware.AuthMiddleware(authClient)) // Use the new middleware
	{
		notificationRoutes.GET("", handlers.GetNotifications)
		notificationRoutes.POST("/:id/read", handlers.MarkNotificationRead)
		notificationRoutes.POST("/read-all", handlers.MarkAllNotificationsRead)
	}

	orderRoutes := r.Group("/orders")
	orderRoutes.Use(middleware.AuthMiddleware(authClient)) // Use the new middleware
	{
		orderRoutes.GET("", handlers.GetBuyerOrders)               // Use orderHandler
		orderRoutes.POST("", handlers.CreateOrder)                 // Use orderHandler
		orderRoutes.POST("/:id/cancel", handlers.CancelOrderBuyer) // Use orderHandler
		orderRoutes.POST("/:id/received", handlers.BuyerReceived)  // Use orderHandler
	}

	// New Delivery Routes
	deliveryRoutes := r.Group("/delivery")
	deliveryRoutes.Use(middleware.AuthMiddleware(authClient)) // Protect delivery estimation
	{
		deliveryRoutes.POST("/intercity/estimate", orderHandler.EstimateIntercityDelivery) // New intercity estimate endpoint
	}

	chatRoutes := r.Group("/chats")
	chatRoutes.Use(middleware.AuthMiddleware(authClient)) // Use the new middleware
	{
		chatRoutes.POST("", handlers.InitiateChat)
		chatRoutes.GET("", handlers.GetChats)
		chatRoutes.POST("/:chat_id/messages", handlers.SendMessage)
		chatRoutes.GET("/:chat_id/messages", handlers.GetChatMessages)
		chatRoutes.DELETE("/:chat_id", handlers.DeleteChat)
		chatRoutes.GET("/upload-url", handlers.GetUploadURL)
	}

	// Admin Routes
	SetupAdminRoutes(r, adminHandler, authClient)
}
