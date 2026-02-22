package models

import "gorm.io/gorm"

type Seller struct {
	gorm.Model
	UserID   uint   `gorm:"unique;not null"`
	User     User   `gorm:"foreignKey:UserID"`
	Status   string `gorm:"default:'pending'"`
	IDCard   string
	Products []Product `gorm:"foreignKey:SellerID"`
}
