package main

import (
	"github.com/gin-gonic/gin"
	"log"
	"ozMadeBack/config"
	"ozMadeBack/internal/auth"
	"ozMadeBack/internal/database"
	"ozMadeBack/internal/routes"
)

func main() {
	config.LoadEnv()
	database.InitDatabase()
	auth.InitFirebase()

	r := gin.Default()
	routes.SetupRoutes(r)

	if err := r.Run(); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
