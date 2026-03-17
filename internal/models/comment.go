package models

import (
	"time"
)

type Comment struct {
	ID        uint `gorm:"primaryKey"`
	ProductID uint
	UserID    uint
	Rating    float64
	Text      string
	CreatedAt time.Time
}
