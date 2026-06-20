package domain

import "time"

type CartItem struct {
	ID string `json:"id" gorm:"primaryKey"`
	CartID    string   `json:"cart_id" gorm:"index"`
	ProductID string `json:"product_id"`
	ProductName string `json:"product_name"`
	Quantity  int32  `json:"quantity"`
	Price     float64 `json:"price"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Cart struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	UserID     string     `json:"user_id" gorm:"index"`
	BusinessID string     `json:"business_id" gorm:"index"`
	Items      []CartItem `json:"items" gorm:"foreignKey:CartID;constraint:OnDelete:CASCADE;"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}
