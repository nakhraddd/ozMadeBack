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
	ProductName     string `gorm:"-" json:"product_name"`
	ProductImage    string `gorm:"-" json:"product_image"`
	SellerName      string `gorm:"-" json:"seller_name"`
	BuyerName       string `gorm:"-" json:"buyer_name"`
	PhoneNumber     string `gorm:"-" json:"phone_number"` // The phone number of the *other* party
	Messages        []Message
}

func (c Chat) ChatIDString() string {
	return fmt.Sprintf("%d_%d_%d", c.BuyerID, c.SellerID, c.ProductID)
}

// MarkAllMessagesAsDeletedForUser marks all messages in the chat as deleted for a specific user.
func (c *Chat) MarkAllMessagesAsDeletedForUser(db *gorm.DB, userID uint, isBuyer bool) error {
	if isBuyer {
		return db.Model(&Message{}).Where("chat_id = ?", c.ID).Update("deleted_by_buyer", true).Error
	}
	return db.Model(&Message{}).Where("chat_id = ?", c.ID).Update("deleted_by_seller", true).Error
}

type Message struct {
	gorm.Model
	ChatID          uint
	SenderID        uint
	SenderRole      string
	Content         string
	CreatedAt       time.Time
	DeletedByBuyer  bool   `gorm:"default:false" json:"deleted_by_buyer"`
	DeletedBySeller bool   `gorm:"default:false" json:"deleted_by_seller"`
	MediaUrl        string `json:"media_url"`
	MediaType       string `json:"media_type"` // photo, audio, video, file
}
