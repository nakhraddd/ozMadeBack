package services

import (
	"context"
	"github.com/go-redis/redis/v8"
	"math"
	"ozMadeBack/internal/database"
	"ozMadeBack/internal/models"
	"time"
)

func StartTrendingWorker() {
	ticker := time.NewTicker(1 * time.Hour)
	for {
		select {
		case <-ticker.C:
			updateTrendingScores()
		}
	}
}

func updateTrendingScores() {
	var products []models.Product
	database.DB.Find(&products)

	for _, p := range products {
		hoursOld := time.Since(p.CreatedAt).Hours()
		score := float64(p.ViewCount) / math.Pow(hoursOld+2, 1.8)

		database.RDB.ZAdd(context.Background(), "trending_products", &redis.Z{
			Score:  score,
			Member: p.ID,
		})
	}
}
