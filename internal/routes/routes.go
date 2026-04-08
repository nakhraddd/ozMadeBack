package routes

import (
	"ozMadeBack/internal/auth"
	"ozMadeBack/internal/handlers"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine, sellerHandler *handlers.SellerHandler) {
	// Public routes
	r.GET("/categories", handlers.GetCategories)
	r.GET("/ads", handlers.GetAds)

	r.GET("/sellers/:id", sellerHandler.GetSellerProfile)
	r.GET("/sellers/:id/reviews", handlers.GetSellerReviews)

	authRoutes := r.Group("/auth")
	authRoutes.Use(auth.AuthMiddleware())
	{
		authRoutes.POST("/sync", handlers.SyncUser)
	}

	productRoutes := r.Group("/products")
	{
		productRoutes.GET("/search", handlers.SearchProducts)
		productRoutes.GET("/trending", handlers.GetTrendingProducts)
		productRoutes.GET("/recommendations", auth.AuthMiddleware(), handlers.GetRecommendations)
		productRoutes.GET("/:id/checkout-options", auth.AuthMiddleware(), handlers.GetCheckoutOptions)
		productRoutes.GET("", handlers.GetProducts)
		productRoutes.GET("/:id", handlers.GetProduct)
		productRoutes.GET("/:id/reviews", handlers.GetProductReviews)
		productRoutes.POST("/:id/view", handlers.ViewProduct)

		productRoutes.Use(auth.AuthMiddleware())
		productRoutes.POST("/:id/comments", handlers.PostComment)
		productRoutes.POST("/:id/report", handlers.ReportProduct)
	}

	userRoutes := r.Group("/profile")
	userRoutes.Use(auth.AuthMiddleware())
	{
		userRoutes.GET("", handlers.GetProfile)
		userRoutes.PATCH("", handlers.UpdateProfile)
		userRoutes.PATCH("/fcm-token", handlers.UpdateFCMToken)
		userRoutes.POST("/favorites/:id", handlers.ToggleFavorite)
		userRoutes.GET("/favorites", handlers.GetFavorites)
		userRoutes.GET("/orders", handlers.GetBuyerOrders)
	}

	orderRoutes := r.Group("/orders")
	orderRoutes.Use(auth.AuthMiddleware())
	{
		orderRoutes.GET("", handlers.GetBuyerOrders)
		orderRoutes.POST("", handlers.CreateOrder)
		orderRoutes.POST("/:id/cancel", handlers.CancelOrderBuyer)
		orderRoutes.POST("/:id/received", handlers.BuyerReceived)
	}

	chatRoutes := r.Group("/chats")
	chatRoutes.Use(auth.AuthMiddleware())
	{
		chatRoutes.POST("", handlers.InitiateChat)
		chatRoutes.GET("", handlers.GetChats)
		chatRoutes.POST("/:chat_id/messages", handlers.SendMessage)
		chatRoutes.GET("/:chat_id/messages", handlers.GetChatMessages)
		chatRoutes.DELETE("/:chat_id", handlers.DeleteChat)
	}
}
