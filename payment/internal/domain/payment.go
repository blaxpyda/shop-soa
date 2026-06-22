package domain

import "time"

type Provider string

const (
	ProviderUnspecified  Provider = "unspecified"
	ProviderMTNMomo      Provider = "mtn_momo"
	ProviderAirtelMoney  Provider = "airtel_money"
)

type PaymentStatus string

const (
	PaymentStatusUnspecified PaymentStatus = "unspecified"
	PaymentStatusPending     PaymentStatus = "pending"
	PaymentStatusSucceeded   PaymentStatus = "succeeded"
	PaymentStatusFailed      PaymentStatus = "failed"
	PaymentStatusRefunded    PaymentStatus = "refunded"
)

type Payment struct {
	ID             string        `gorm:"primaryKey"`
	OrderID        string        `gorm:"not null;index"`
	UserID         string        `gorm:"not null;index"`
	Amount         int64         `gorm:"not null"`
	Currency       string        `gorm:"not null"`
	Provider       Provider      `gorm:"not null"`
	ProviderRef    string        // set when provider confirms
	Phone          string        `gorm:"not null"`
	Status         PaymentStatus `gorm:"not null"`
	IdempotencyKey string        `gorm:"uniqueIndex"`
	CreatedAt      time.Time     `gorm:"autoCreateTime"`
	UpdatedAt      time.Time     `gorm:"autoUpdateTime"`
}

type TransactionType string

const (
	TransactionTypeUnspecified TransactionType = "unspecified"
	TransactionTypePayment     TransactionType = "payment"
	TransactionTypeCommission  TransactionType = "commission"
	TransactionTypePayout      TransactionType = "payout"
	TransactionTypeRefund      TransactionType = "refund"
)

// Transaction is one immutable ledger entry. business_id="" means platform.
type Transaction struct {
	ID         string          `gorm:"primaryKey"`
	PaymentID  string          `gorm:"index"`  // "" for payouts
	BusinessID string          `gorm:"index"`
	Type       TransactionType `gorm:"not null"`
	Amount     int64           `gorm:"not null"` // always positive
	Currency   string          `gorm:"not null"`
	Note       string
	CreatedAt  time.Time `gorm:"autoCreateTime"`
}

type Payout struct {
	ID             string        `gorm:"primaryKey"`
	BusinessID     string        `gorm:"not null;index"`
	Amount         int64         `gorm:"not null"`
	Currency       string        `gorm:"not null"`
	Provider       Provider      `gorm:"not null"`
	ProviderRef    string
	Phone          string        `gorm:"not null"`
	Status         PaymentStatus `gorm:"not null"`
	IdempotencyKey string        `gorm:"uniqueIndex"`
	CreatedAt      time.Time     `gorm:"autoCreateTime"`
	UpdatedAt      time.Time     `gorm:"autoUpdateTime"`
}

// ---- Input / filter types ----

type InitiatePaymentInput struct {
	OrderID        string
	UserID         string
	Amount         int64
	Currency       string
	Provider       Provider
	Phone          string
	IdempotencyKey string
}

type RefundInput struct {
	PaymentID      string
	Amount         int64 // 0 = full refund
	Reason         string
	IdempotencyKey string
}

type ListTransactionsFilter struct {
	BusinessID string
	PaymentID  string
	PageSize   int
	PageToken  string
}

type InitiatePayoutInput struct {
	BusinessID     string
	Amount         int64
	Currency       string
	Provider       Provider
	Phone          string
	IdempotencyKey string
}
