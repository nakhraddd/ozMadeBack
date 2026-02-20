package models

type Favorite struct {
	UserID    uint `gorm:"primaryKey"`
	ProductID uint `gorm:"primaryKey"`
}
