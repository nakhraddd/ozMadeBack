package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetCategories(c *gin.Context) {
	categories := []gin.H{
		{"id": "food", "title": "Еда", "icon_url": nil},
		{"id": "art", "title": "Искусство", "icon_url": nil},
		{"id": "clothing", "title": "Одежда", "icon_url": nil},
		{"id": "electronics", "title": "Электроника", "icon_url": nil},
		{"id": "home", "title": "Дом", "icon_url": nil},
	}
	c.JSON(http.StatusOK, categories)
}

func GetAds(c *gin.Context) {
	ads := []gin.H{
		{
			"id":        "1",
			"image_url": "https://storage.googleapis.com/ozmade-bucket/ads/ad1.jpg", // Example URL
			"title":     "Скидки",
			"deeplink":  "ozmade://discounts",
		},
	}
	c.JSON(http.StatusOK, ads)
}
