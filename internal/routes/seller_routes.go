package routes

import (
	"ozMadeBack/internal/handlers"
	"ozMadeBack/internal/middleware"

	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
)

func SellerRoutes(r *gin.Engine, sellerHandler *handlers.SellerHandler, authClient *auth.Client) {
	sellerGroup := r.Group("/seller")
	sellerGroup.Use(middleware.AuthMiddleware(authClient))
	{
		sellerGroup.POST("/register", sellerHandler.RegisterSeller)
		sellerGroup.GET("/upload-id-url", sellerHandler.GetUploadIDURL)
		sellerGroup.GET("/upload-product-photo-url", sellerHandler.GetUploadProductPhotoURL)

		sellerGroup.Use(middleware.SellerMiddleware())
		{
			sellerGroup.GET("/products", sellerHandler.GetProducts)
			sellerGroup.POST("/products", sellerHandler.CreateProduct)
			sellerGroup.PUT("/products/:id", sellerHandler.UpdateProduct)
			sellerGroup.DELETE("/products/:id", sellerHandler.DeleteProduct)

			sellerGroup.GET("/profile", sellerHandler.GetProfile)
			sellerGroup.PATCH("/profile", sellerHandler.UpdateProfile)

			sellerGroup.GET("/delivery", sellerHandler.GetDelivery)
			sellerGroup.PATCH("/delivery", sellerHandler.UpdateDelivery)

			sellerGroup.GET("/orders", sellerHandler.GetSellerOrders)
			sellerGroup.POST("/orders/:id/confirm", sellerHandler.ConfirmOrder)
			sellerGroup.POST("/orders/:id/cancel", sellerHandler.CancelOrderSeller)
			sellerGroup.POST("/orders/:id/ready_or_shipped", sellerHandler.ReadyOrShipped)
			sellerGroup.POST("/orders/:id/complete", sellerHandler.CompleteOrder)

			sellerGroup.GET("/chats", handlers.GetChats)
			sellerGroup.GET("/chats/:chat_id/messages", handlers.GetChatMessages)
			sellerGroup.POST("/chats/:chat_id/messages", handlers.SendMessage)
		}
	}
}
