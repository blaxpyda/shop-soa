package internal

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"thugcorp.io/grocery/payment/internal/domain"
)

type PaymentRepository interface {
	Create(ctx context.Context, input domain.CreatePaymentInput) (*domain.Payment, error)
	GetByID(ctx context.Context, id string) (*domain.Payment, error)
	Update(ctx context.Context, id string, input domain.UpdatePaymentInput) (*domain.Payment, error)
	List(ctx context.Context, filter domain.ListFilter) ([]*domain.Payment, int64, error)
}

type paymentRepository struct {
	db *gorm.DB
}

func NewPaymentRepository(db *gorm.DB) PaymentRepository {
	return &paymentRepository{db: db}
}

func (r *paymentRepository) Create(ctx context.Context, input domain.CreatePaymentInput) (*domain.Payment, error) {
	meta := "{}"
	if len(input.Metadata) > 0 {
		b, err := json.Marshal(input.Metadata)
		if err != nil {
			return nil, err
		}
		meta = string(b)
	}

	p := &domain.Payment{
		ID:            uuid.NewString(),
		OrderID:       input.OrderID,
		UserID:        input.UserID,
		BusinessID:    input.BusinessID,
		Amount:        input.Amount,
		Currency:      input.Currency,
		Status:        domain.StatusPending,
		PaymentMethod: input.PaymentMethod,
		Provider:      input.Provider,
		Metadata:      meta,
	}

	if err := r.db.WithContext(ctx).Create(p).Error; err != nil {
		return nil, err
	}
	return p, nil
}

func (r *paymentRepository) GetByID(ctx context.Context, id string) (*domain.Payment, error) {
	var p domain.Payment
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &p, err
}

func (r *paymentRepository) Update(ctx context.Context, id string, input domain.UpdatePaymentInput) (*domain.Payment, error) {
	updates := map[string]any{}
	if input.Status != nil {
		updates["status"] = *input.Status
	}
	if input.TransactionRef != nil {
		updates["transaction_ref"] = *input.TransactionRef
	}
	if input.ErrorMessage != nil {
		updates["error_message"] = *input.ErrorMessage
	}

	if err := r.db.WithContext(ctx).Model(&domain.Payment{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return nil, err
	}
	return r.GetByID(ctx, id)
}

func (r *paymentRepository) List(ctx context.Context, filter domain.ListFilter) ([]*domain.Payment, int64, error) {
	query := r.db.WithContext(ctx).Model(&domain.Payment{})

	if filter.UserID != "" {
		query = query.Where("user_id = ?", filter.UserID)
	}
	if filter.BusinessID != "" {
		query = query.Where("business_id = ?", filter.BusinessID)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	pageSize := int(filter.PageSize)
	if pageSize <= 0 {
		pageSize = 20
	}
	page := int(filter.Page)
	if page <= 0 {
		page = 1
	}

	var payments []*domain.Payment
	err := query.Order("created_at DESC").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&payments).Error
	return payments, total, err
}
