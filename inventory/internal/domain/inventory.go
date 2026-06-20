package domain

import "time"

// StockItem is keyed by (business_id, product_id, location_id).
// available is derived (on_hand - reserved) and never stored.
type StockItem struct {
	BusinessID        string    `gorm:"primaryKey"`
	ProductID         string    `gorm:"primaryKey"`
	LocationID        string    `gorm:"primaryKey;default:''"`
	OnHand            int64     `gorm:"not null;default:0"`
	Reserved          int64     `gorm:"not null;default:0"`
	LowStockThreshold int64     `gorm:"not null;default:0"`
	State             string    `gorm:"not null;default:'STOCK_STATE_IN_STOCK'"`
	Version           string    `gorm:"not null"` // UUID, bumped on every write
	UpdatedAt         time.Time `gorm:"autoUpdateTime"`
}

func (s *StockItem) Available() int64 {
	return s.OnHand - s.Reserved
}

// Reservation holds stock across one or more lines for an order.
type Reservation struct {
	ID             string            `gorm:"primaryKey;type:uuid"`
	OrderID        string            `gorm:"not null;index"`
	Status         string            `gorm:"not null"` // HELD | COMMITTED | RELEASED
	ExpiresAt      time.Time         `gorm:"not null;index"`
	IdempotencyKey string            `gorm:"uniqueIndex"`
	CreatedAt      time.Time         `gorm:"autoCreateTime"`
	Items          []ReservationItem `gorm:"foreignKey:ReservationID;constraint:OnDelete:CASCADE"`
}

// ReservationItem is a single stock line within a Reservation.
type ReservationItem struct {
	ID            string `gorm:"primaryKey;type:uuid"`
	ReservationID string `gorm:"not null;index"`
	BusinessID    string `gorm:"not null"`
	ProductID     string `gorm:"not null"`
	LocationID    string `gorm:"not null;default:''"`
	Quantity      int64  `gorm:"not null"`
}

// ShortfallResult is returned when a line cannot be fully reserved.
type ShortfallResult struct {
	ProductID  string
	LocationID string
	Available  int64
	State      string
}

// ComputeState derives stock state from current on_hand/reserved values.
// DISCONTINUED is never overwritten by this function.
func ComputeState(item *StockItem) string {
	if item.State == "STOCK_STATE_DISCONTINUED" {
		return "STOCK_STATE_DISCONTINUED"
	}
	avail := item.Available()
	if avail <= 0 {
		return "STOCK_STATE_OUT_OF_STOCK"
	}
	if item.LowStockThreshold > 0 && avail <= item.LowStockThreshold {
		return "STOCK_STATE_LOW"
	}
	return "STOCK_STATE_IN_STOCK"
}
