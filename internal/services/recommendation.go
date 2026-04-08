package services

import (
	"context"
	"log"
	recommendationservice "ozMadeBack/internal/service/recommendation"
	"time"
)

const recommendationRefreshLimit = 50

func StartRecommendationWorker() {
	service := recommendationservice.NewDefaultService()

	if err := service.RefreshGlobalRecommendations(context.Background(), recommendationRefreshLimit); err != nil {
		log.Printf("failed to build initial global recommendations: %v", err)
	}

	if err := service.RefreshAllUserRecommendations(context.Background(), recommendationRefreshLimit); err != nil {
		log.Printf("failed to build initial user recommendations: %v", err)
	}

	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := service.RefreshGlobalRecommendations(context.Background(), recommendationRefreshLimit); err != nil {
				log.Printf("failed to refresh global recommendations: %v", err)
			}
			if err := service.RefreshAllUserRecommendations(context.Background(), recommendationRefreshLimit); err != nil {
				log.Printf("failed to refresh user recommendations: %v", err)
			}
		}
	}
}
