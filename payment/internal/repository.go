package internal

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"thugcorp.io/payment/internal/domain"
)

type PaymentRepository interface {
	CreatePayment(p *domain.Payment) error
	GetPaymentByID(id string) (*domain.Payment, error)
	GetPaymentByIdempotencyKey(key string) (*domain.Payment, error)
	UpdatePaymentStatus(id string, status domain.PaymentStatus, providerRef string) error

	CreateTransaction(t *domain.Transaction) error
	ListTransactions(filter domain.ListTransactionsFilter) ([]domain.Transaction, string, error)
	SumTransactionsByBusiness(businessID string) (map[domain.TransactionType]int64, string, error)

	CreatePayout(p *domain.Payout) error
	GetPayoutByIdempotencyKey(key string) (*domain.Payout, error)
	UpdatePayoutStatus(id string, status domain.PaymentStatus, providerRef string) error
}

type paymentRepository struct {
	db *gorm.DB
}

func NewPaymentRepository(db *gorm.DB) PaymentRepository {
	return &paymentRepository{db: db}
}

func (r *paymentRepository) CreatePayment(p *domain.Payment) error {
	return r.db.Create(p).Error
}

func (r *paymentRepository) GetPaymentByID(id string) (*domain.Payment, error) {
	var p domain.Payment
	err := r.db.Where("id = ?", id).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("payment not found")
	}
	return &p, err
}

func (r *paymentRepository) GetPaymentByIdempotencyKey(key string) (*domain.Payment, error) {
	var p domain.Payment
	err := r.db.Where("idempotency_key = ?", key).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &p, err
}

func (r *paymentRepository) UpdatePaymentStatus(id string, status domain.PaymentStatus, providerRef string) error {
	return r.db.Model(&domain.Payment{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":       status,
			"provider_ref": providerRef,
			"updated_at":   time.Now(),
		}).Error
}

func (r *paymentRepository) CreateTransaction(t *domain.Transaction) error {
	return r.db.Create(t).Error
}

func (r *paymentRepository) ListTransactions(filter domain.ListTransactionsFilter) ([]domain.Transaction, string, error) {
	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	offset := 0
	if filter.PageToken != "" {
		decoded, err := base64.StdEncoding.DecodeString(filter.PageToken)
		if err == nil {
			offset, _ = strconv.Atoi(string(decoded))
		}
	}

	query := r.db.Model(&domain.Transaction{})
	if filter.BusinessID != "" {
		query = query.Where("business_id = ?", filter.BusinessID)
	}
	if filter.PaymentID != "" {
		query = query.Where("payment_id = ?", filter.PaymentID)
	}

	var txns []domain.Transaction
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&txns).Error; err != nil {
		return nil, "", err
	}

	nextToken := ""
	if len(txns) == pageSize {
		nextOffset := offset + pageSize
		nextToken = base64.StdEncoding.EncodeToString([]byte(strconv.Itoa(nextOffset)))
	}
	return txns, nextToken, nil
}

func (r *paymentRepository) SumTransactionsByBusiness(businessID string) (map[domain.TransactionType]int64, string, error) {
	type row struct {
		Type   domain.TransactionType
		Total  int64
	}
	var rows []row
	err := r.db.Model(&domain.Transaction{}).
		Select("type, SUM(amount) as total").
		Where("business_id = ?", businessID).
		Group("type").
		Scan(&rows).Error
	if err != nil {
		return nil, "", err
	}

	sums := make(map[domain.TransactionType]int64)
	currency := ""
	for _, row := range rows {
		sums[row.Type] = row.Total
	}

	// fetch currency from the first transaction
	var first domain.Transaction
	if err := r.db.Where("business_id = ?", businessID).First(&first).Error; err == nil {
		currency = first.Currency
	}
	return sums, currency, nil
}

func (r *paymentRepository) CreatePayout(p *domain.Payout) error {
	return r.db.Create(p).Error
}

func (r *paymentRepository) GetPayoutByIdempotencyKey(key string) (*domain.Payout, error) {
	var p domain.Payout
	err := r.db.Where("idempotency_key = ?", key).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &p, err
}

func (r *paymentRepository) UpdatePayoutStatus(id string, status domain.PaymentStatus, providerRef string) error {
	return r.db.Model(&domain.Payout{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":       status,
			"provider_ref": providerRef,
			"updated_at":   time.Now(),
		}).Error
}

func newID() string { return uuid.NewString() }
