package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

func LoadEnv() {
	// We ignore the error because in Docker/Production,
	// variables are often provided by the environment, not a file.
	if err := godotenv.Load(); err != nil {
		log.Println("Note: Using system environment variables (no .env file found)")
	}
}

func GetEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}
