package internal

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"thugcorp.io/grocery/transaction/internal/domain"
)

type TransactionRepository interface {
	Create(ctx context.Context, t *domain.Transaction) (*domain.Transaction, error)
	GetByID(ctx context.Context, id string) (*domain.Transaction, error)
	UpdateStatus(ctx context.Context, id, status string) (*domain.Transaction, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, userID, businessID, status string, page, pageSize int) ([]*domain.Transaction, int64, error)
}

type transactionRepository struct {
	db *gorm.DB
}

func NewTransactionRepository(db *gorm.DB) TransactionRepository {
	return &transactionRepository{db: db}
}

func (r *transactionRepository) Create(ctx context.Context, t *domain.Transaction) (*domain.Transaction, error) {
	t.ID = uuid.New().String()
	for i := range t.Items {
		t.Items[i].ID = uuid.New().String()
		t.Items[i].TransactionID = t.ID
	}
	if err := r.db.WithContext(ctx).Create(t).Error; err != nil {
		return nil, err
	}
	return r.GetByID(ctx, t.ID)
}

func (r *transactionRepository) GetByID(ctx context.Context, id string) (*domain.Transaction, error) {
	var t domain.Transaction
	err := r.db.WithContext(ctx).Preload("Items").Where("id = ?", id).First(&t).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

func (r *transactionRepository) UpdateStatus(ctx context.Context, id, status string) (*domain.Transaction, error) {
	if err := r.db.WithContext(ctx).Model(&domain.Transaction{}).
		Where("id = ?", id).
		Update("status", status).Error; err != nil {
		return nil, err
	}
	return r.GetByID(ctx, id)
}

func (r *transactionRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&domain.Transaction{}).Error
}

func (r *transactionRepository) List(ctx context.Context, userID, businessID, status string, page, pageSize int) ([]*domain.Transaction, int64, error) {
	if pageSize <= 0 {
		pageSize = 20
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * pageSize

	q := r.db.WithContext(ctx).Model(&domain.Transaction{})
	if userID != "" {
		q = q.Where("user_id = ?", userID)
	}
	if businessID != "" {
		q = q.Where("business_id = ?", businessID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []*domain.Transaction
	if err := q.Preload("Items").Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}
