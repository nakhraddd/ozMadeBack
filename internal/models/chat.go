package models

import (
	"gorm.io/gorm"
	"time"
)

type Chat struct {
	gorm.Model
	SellerID uint
	BuyerID  uint
	Messages []Message
}

type Message struct {
	gorm.Model
	ChatID    uint
	SenderID  uint
	Content   string
	CreatedAt time.Time
}
