package handlers

import (
	"net/http"
	"ozMadeBack/internal/dto"

	"github.com/gin-gonic/gin"
)

func GetCategories(c *gin.Context) {
	categories := []dto.CategoryDto{
		{ID: "food", Title: "Еда", IconURL: nil},
		{ID: "art", Title: "Искусство", IconURL: nil},
		{ID: "clothing", Title: "Одежда", IconURL: nil},
		{ID: "electronics", Title: "Электроника", IconURL: nil},
		{ID: "home", Title: "Дом", IconURL: nil},
	}
	c.JSON(http.StatusOK, categories)
}

func GetAds(c *gin.Context) {
	ads := []dto.AdDto{
		{
			ID:       "1",
			ImageURL: "https://storage.googleapis.com/ozmade-bucket/ads/ad1.jpg",
			Title:    "Скидки",
			Deeplink: "ozmade://discounts",
		},
	}
	c.JSON(http.StatusOK, ads)
}
