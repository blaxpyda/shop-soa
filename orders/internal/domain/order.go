package domain

import "time"

type Order struct {
	ID              string      `gorm:"primaryKey;type:uuid" json:"id"`
	UserID          string      `gorm:"not null;index" json:"user_id"`
	BusinessID      string      `gorm:"not null;index" json:"business_id"`
	Items           []OrderItem `gorm:"foreignKey:OrderID;constraint:OnDelete:CASCADE" json:"items"`
	TotalAmount     float64     `gorm:"not null;default:0" json:"total_amount"`
	Status          string      `gorm:"not null;default:'pending'" json:"status"`
	ShippingAddress string      `json:"shipping_address"`
	PaymentMethod   string      `json:"payment_method"`
	CancelReason    string      `json:"cancel_reason,omitempty"`
	CreatedAt       time.Time   `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time   `gorm:"autoUpdateTime" json:"updated_at"`
}

type OrderItem struct {
	ID          string  `gorm:"primaryKey;type:uuid" json:"id"`
	OrderID     string  `gorm:"not null;index" json:"order_id"`
	ProductID   string  `gorm:"not null" json:"product_id"`
	ProductName string  `gorm:"not null" json:"product_name"`
	Quantity    int32   `gorm:"not null" json:"quantity"`
	Price       float64 `gorm:"not null" json:"price"`
}

type CreateOrderInput struct {
	UserID          string
	BusinessID      string
	ShippingAddress string
	PaymentMethod   string
}

type UpdateOrderStatusInput struct {
	Status string
}
