package routes

import (
	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
	"ozMadeBack/internal/handlers"
	"ozMadeBack/internal/middleware"
)

func SellerRoutes(r *gin.Engine, sellerHandler *handlers.SellerHandler, authClient *auth.Client) {
	sellerGroup := r.Group("/seller")
	sellerGroup.Use(middleware.AuthMiddleware(authClient))

	sellerGroup.POST("/register", sellerHandler.RegisterSeller)
	sellerGroup.GET("/upload-id-url", sellerHandler.GetUploadIDURL)

	sellerGroup.Use(middleware.SellerMiddleware())
	{
		sellerGroup.GET("/products", sellerHandler.GetProducts)
		sellerGroup.POST("/products", sellerHandler.CreateProduct)
		sellerGroup.PUT("/products/:id", sellerHandler.UpdateProduct)
		sellerGroup.DELETE("/products/:id", sellerHandler.DeleteProduct)

		sellerGroup.GET("/profile", sellerHandler.GetProfile)
		sellerGroup.PATCH("/profile", sellerHandler.UpdateProfile)

		sellerGroup.GET("/chats", sellerHandler.GetChats)
		sellerGroup.GET("/chats/:chat_id/messages", sellerHandler.GetChatMessages)
	}
}
