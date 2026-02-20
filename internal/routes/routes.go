package routes

import (
	"github.com/gin-gonic/gin"
	"ozMadeBack/internal/auth"
	"ozMadeBack/internal/handlers"
)

func SetupRoutes(r *gin.Engine) {
	authRoutes := r.Group("/auth")
	authRoutes.Use(auth.AuthMiddleware())
	{
		authRoutes.POST("/sync", handlers.SyncUser)
	}

	productRoutes := r.Group("/products")
	{
		productRoutes.GET("", handlers.GetProducts)
		productRoutes.GET("/:id", handlers.GetProduct)
		productRoutes.POST("/:id/view", handlers.ViewProduct)
		productRoutes.GET("/trending", handlers.GetTrendingProducts)

		productRoutes.Use(auth.AuthMiddleware())
		productRoutes.POST("/:id/comments", handlers.PostComment)
		productRoutes.POST("/:id/report", handlers.ReportProduct)
	}

	userRoutes := r.Group("/profile")
	userRoutes.Use(auth.AuthMiddleware())
	{
		userRoutes.GET("", handlers.GetProfile)
		userRoutes.PATCH("", handlers.UpdateProfile)
		userRoutes.POST("/favorites/:id", handlers.ToggleFavorite)
		userRoutes.GET("/favorites", handlers.GetFavorites)
		userRoutes.GET("/orders", handlers.GetOrders)
	}
}
