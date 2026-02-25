package models

import (
	"time"

	"gorm.io/gorm"
)

type Chat struct {
	gorm.Model
	SellerID     uint
	BuyerID      uint
	ProductID    uint   // New field to link chat to a product
	ProductName  string `gorm:"-"` // Populated at runtime
	ProductImage string `gorm:"-"` // Populated at runtime
	Messages     []Message
}

type Message struct {
	gorm.Model
	ChatID     uint
	SenderID   uint
	SenderRole string // "SELLER" or "BUYER"
	Content    string
	CreatedAt  time.Time
}
