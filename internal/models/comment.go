package models

import "gorm.io/gorm"

type Comment struct {
	gorm.Model
	ProductID uint
	UserID    uint
	User      User    `gorm:"foreignKey:UserID"`
	Product   Product `gorm:"foreignKey:ProductID"`
	Rating    float64
	Text      string
}
