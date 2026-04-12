package models

import "gorm.io/gorm"

type Comment struct {
	gorm.Model
	ProductID uint    `json:"product_id"`
	UserID    uint    `json:"user_id"`
	User      User    `gorm:"foreignKey:UserID" json:"user"`
	Product   Product `gorm:"foreignKey:ProductID" json:"product"`
	Rating    float64 `json:"rating"`
	Text      string  `json:"text"`
}
