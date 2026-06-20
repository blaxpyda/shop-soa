package domain

import (
	"errors"
	"time"
)

const (
	StatusPending   = "pending"
	StatusCompleted = "completed"
	StatusFailed    = "failed"
	StatusRefunded  = "refunded"
)

var (
	ErrNotFound     = errors.New("payment not found")
	ErrInvalidInput = errors.New("invalid input")
	ErrInvalidState = errors.New("invalid state")
)

type Payment struct {
	ID             string    `gorm:"primaryKey"`
	OrderID        string    `gorm:"index"`
	UserID         string    `gorm:"index"`
	BusinessID     string    `gorm:"index"`
	Amount         float64
	Currency       string
	Status         string `gorm:"index"`
	PaymentMethod  string
	Provider       string
	TransactionRef string
	ErrorMessage   string
	Metadata       string // JSON-encoded map[string]string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type CreatePaymentInput struct {
	OrderID       string
	UserID        string
	BusinessID    string
	Amount        float64
	Currency      string
	PaymentMethod string
	Provider      string
	Metadata      map[string]string
}

type UpdatePaymentInput struct {
	Status         *string
	TransactionRef *string
	ErrorMessage   *string
}

type ListFilter struct {
	UserID     string
	BusinessID string
	Status     string
	Page       int32
	PageSize   int32
}
