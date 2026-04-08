package models

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type Chat struct {
	gorm.Model
	SellerID        uint   `gorm:"uniqueIndex:idx_chat_unique"`
	BuyerID         uint   `gorm:"uniqueIndex:idx_chat_unique"`
	ProductID       uint   `gorm:"uniqueIndex:idx_chat_unique"`
	DeletedByBuyer  bool   `gorm:"default:false" json:"deleted_by_buyer"`
	DeletedBySeller bool   `gorm:"default:false" json:"deleted_by_seller"`
	ProductName     string `gorm:"-"`
	ProductImage    string `gorm:"-"`
	Messages        []Message
}

func (c Chat) ChatIDString() string {
	return fmt.Sprintf("%d_%d_%d", c.BuyerID, c.SellerID, c.ProductID)
}

type Message struct {
	gorm.Model
	ChatID     uint
	SenderID   uint
	SenderRole string
	Content    string
	CreatedAt  time.Time
}
