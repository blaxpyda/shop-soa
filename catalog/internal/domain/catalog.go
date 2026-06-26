package domain

import "time"

// Product holds catalog definition + price (slow-changing data).
type Product struct {
	ID          string `gorm:"primaryKey"`
	BusinessID  string `gorm:"not null;index"`
	Name        string `gorm:"not null"`
	Description string
	Category    string `gorm:"index"`
	Price       int64  `gorm:"not null"`
	CostPrice   int64  `gorm:"not null;default:0"`
	Currency    string `gorm:"not null"`
	ImageURL    string
	Active      bool      `gorm:"not null;default:true"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}

// StockItem tracks on-hand and reserved counts per product/location.
// LocationID is "" for single-location businesses.
type StockItem struct {
	ProductID  string    `gorm:"primaryKey"`
	BusinessID string    `gorm:"not null;index"`
	OnHand     int64     `gorm:"not null;default:0"`
	Reserved   int64     `gorm:"not null;default:0"`
	Version    string    `gorm:"not null"` // UUID rotated on each write for optimistic concurrency
	UpdatedAt  time.Time `gorm:"autoUpdateTime"`
}

func (s *StockItem) Available() int64 {
	if avail := s.OnHand - s.Reserved; avail > 0 {
		return avail
	}
	return 0
}

// AdjustmentLog deduplicates AdjustStock calls by idempotency key.
type AdjustmentLog struct {
	IdempotencyKey string    `gorm:"primaryKey"`
	ProductID      string    `gorm:"not null"`
	UnitCost       int64     // purchase price per unit in minor units (RESTOCK only)
	CreatedAt      time.Time `gorm:"autoCreateTime"`
}

type ReservationStatus string

const (
	ReservationStatusPending   ReservationStatus = "pending"
	ReservationStatusCommitted ReservationStatus = "committed"
	ReservationStatusReleased  ReservationStatus = "released"
)

type Reservation struct {
	ID             string            `gorm:"primaryKey"`
	OrderID        string            `gorm:"not null;index"`
	IdempotencyKey string            `gorm:"uniqueIndex"`
	Status         ReservationStatus `gorm:"not null;default:'pending'"`
	ExpiresAt      time.Time         `gorm:"not null"`
	CreatedAt      time.Time         `gorm:"autoCreateTime"`
	Items          []ReservationItem `gorm:"foreignKey:ReservationID;constraint:OnDelete:CASCADE"`
}

type ReservationItem struct {
	ID            string `gorm:"primaryKey"`
	ReservationID string `gorm:"not null;index"`
	ProductID     string `gorm:"not null"`
	LocationID    string
	Quantity      int64 `gorm:"not null"`
}

// ---- Input / filter types ----

type CreateProductInput struct {
	BusinessID   string
	Name         string
	Description  string
	Category     string
	Price        int64
	CostPrice    int64
	Currency     string
	ImageURL     string
	InitialStock int64
}

type UpdateProductInput struct {
	Name        string
	Description string
	Category    string
	Price       int64
	CostPrice   int64
	Active      *bool
}

type ListProductsFilter struct {
	BusinessID string
	Query      string // optional name/description substring
	PageSize   int
	PageToken  string // encoded offset cursor
}

type AdjustStockInput struct {
	ProductID       string
	LocationID      string
	Delta           *int64 // mutually exclusive with SetTo
	SetTo           *int64
	Reason          string
	ExpectedVersion string
	IdempotencyKey  string
	UnitCost        int64 // purchase price per unit in minor units (RESTOCK only)
}

type AvailabilityQuery struct {
	ProductID  string
	LocationID string
	Quantity   int64
}

type AvailabilityResult struct {
	ProductID  string
	Available  int64
	Sufficient bool
}

type ReserveInput struct {
	OrderID        string
	Items          []ReserveItemInput
	TTL            time.Duration
	IdempotencyKey string
}

type ReserveItemInput struct {
	ProductID  string
	LocationID string
	Quantity   int64
}
