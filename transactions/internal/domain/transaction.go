package domain

import "time"

type Transaction struct {
	ID            string            `gorm:"primaryKey;type:uuid"`
	UserID        string            `gorm:"not null;index"`
	BusinessID    string            `gorm:"not null;index"`
	TotalAmount   float64           `gorm:"not null;default:0"`
	Status        string            `gorm:"not null;default:'pending'"`
	PaymentMethod string            `gorm:"not null;default:''"`
	Items         []TransactionItem `gorm:"foreignKey:TransactionID;constraint:OnDelete:CASCADE"`
	CreatedAt     time.Time         `gorm:"autoCreateTime"`
	UpdatedAt     time.Time         `gorm:"autoUpdateTime"`
}

type TransactionItem struct {
	ID            string  `gorm:"primaryKey;type:uuid"`
	TransactionID string  `gorm:"not null;index"`
	ProductID     string  `gorm:"not null"`
	ProductName   string  `gorm:"not null;default:''"`
	BusinessID    string  `gorm:"not null"`
	Quantity      int32   `gorm:"not null;default:0"`
	Price         float64 `gorm:"not null;default:0"`
}
