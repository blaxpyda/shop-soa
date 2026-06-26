package domain

import "time"

// ---- Cart ----

// Cart is keyed by UserID; it exists implicitly once the first item is added.
type Cart struct {
	UserID    string     `gorm:"primaryKey"`
	Currency  string
	UpdatedAt time.Time  `gorm:"autoUpdateTime"`
	Items     []CartItem `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}

// CartItem caches name, price, and currency from the catalog so the UI can
// render the cart without a catalog round-trip on every GET.
type CartItem struct {
	UserID     string `gorm:"primaryKey"`
	ProductID  string `gorm:"primaryKey"`
	BusinessID string `gorm:"not null"`
	Name       string `gorm:"not null"`
	UnitPrice  int64  `gorm:"not null"`
	Currency   string `gorm:"not null"`
	Quantity   int64  `gorm:"not null"`
}

func (c *Cart) Subtotal() int64 {
	var total int64
	for _, item := range c.Items {
		total += item.UnitPrice * item.Quantity
	}
	return total
}

// DisplayCurrency returns the currency of the first cart item, or the cart's
// stored currency, for use in proto responses.
func (c *Cart) DisplayCurrency() string {
	if c.Currency != "" {
		return c.Currency
	}
	if len(c.Items) > 0 {
		return c.Items[0].Currency
	}
	return ""
}

// ---- Orders ----

type OrderStatus string

const (
	OrderStatusUnspecified    OrderStatus = "unspecified"
	OrderStatusPendingPayment OrderStatus = "pending_payment"
	OrderStatusConfirmed      OrderStatus = "confirmed"
	OrderStatusPreparing      OrderStatus = "preparing"
	OrderStatusOutForDelivery OrderStatus = "out_for_delivery"
	OrderStatusDelivered      OrderStatus = "delivered"
	OrderStatusCancelled      OrderStatus = "cancelled"
)

type Order struct {
	ID              string      `gorm:"primaryKey"`
	UserID          string      `gorm:"not null;index"`
	Status          OrderStatus `gorm:"not null"`
	Total           int64       `gorm:"not null"`
	Currency        string      `gorm:"not null"`
	PaymentID       string
	DeliveryAddress string
	IdempotencyKey  string      `gorm:"uniqueIndex"`
	CreatedAt       time.Time   `gorm:"autoCreateTime"`
	UpdatedAt       time.Time   `gorm:"autoUpdateTime"`
	Items           []OrderItem `gorm:"foreignKey:OrderID;constraint:OnDelete:CASCADE"`
}

// OrderItem snapshots name and price at checkout so the order is a permanent
// correct record even after the catalog changes.
type OrderItem struct {
	ID          string `gorm:"primaryKey"`
	OrderID     string `gorm:"not null;index"`
	ProductID   string `gorm:"not null"`
	BusinessID  string `gorm:"not null;index"`
	ProductName string `gorm:"not null"`
	UnitPrice   int64  `gorm:"not null"`
	Quantity    int64  `gorm:"not null"`
	LineTotal   int64  `gorm:"not null"`
}

// ---- Input / filter types ----

type CheckoutInput struct {
	UserID          string
	DeliveryAddress string
	PaymentMethod   string
	IdempotencyKey  string
}

type ListOrdersFilter struct {
	UserID     string
	BusinessID string
	Status     OrderStatus
	PageSize   int
	PageToken  string
	AllOrders  bool // super-admin: bypass user/business scoping
}
