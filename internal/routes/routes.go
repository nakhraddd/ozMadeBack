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
}
