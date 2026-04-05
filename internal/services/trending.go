package services

import (
	"context"
	"log"
	productservice "ozMadeBack/internal/service/product"
	"time"
)

func StartTrendingWorker() {
	service := productservice.NewDefaultService()
	if err := service.RefreshTrendingScores(context.Background()); err != nil {
		log.Printf("failed to build initial trending scores: %v", err)
	}

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := service.RefreshTrendingScores(context.Background()); err != nil {
				log.Printf("failed to refresh trending scores: %v", err)
			}
		}
	}
}
