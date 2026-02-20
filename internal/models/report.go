package models

import (
	"time"
)

type Report struct {
	ID        uint `gorm:"primaryKey"`
	UserID    uint
	ProductID uint
	Reason    string
	CreatedAt time.Time
}
