package main

import (
	"log"
	"ozMadeBack/config"
	"ozMadeBack/internal/auth"
	"ozMadeBack/internal/database"
	"ozMadeBack/internal/routes"
	"ozMadeBack/internal/services"

	"github.com/gin-gonic/gin"
)

func main() {
	config.LoadEnv()
	database.InitDatabase()
	database.InitRedis()
	auth.InitFirebase()

	go services.StartTrendingWorker()

	r := gin.Default()
	routes.SetupRoutes(r)

	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
