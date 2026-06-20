package domain

import "time"

type Product struct {
	ID              string    `gorm:"primaryKey;type:uuid"`
	BusinessID      string    `gorm:"not null;index"`
	Name            string    `gorm:"not null"`
	Category        string    `gorm:"not null;default:''"`
	Price           float64   `gorm:"not null;default:0"`
	Quantity        float64   `gorm:"not null;default:0"`
	ImageURL        string    `gorm:"not null;default:''"`
	IsActive        bool      `gorm:"not null;default:true"`
	CreatedAt       time.Time `gorm:"autoCreateTime"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime"`
}
