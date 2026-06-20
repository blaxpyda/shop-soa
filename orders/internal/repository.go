package internal

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"thugcorp.io/grocery/order/internal/domain"
)

type OrderRepository interface {
	Create(ctx context.Context, input domain.CreateOrderInput) (*domain.Order, error)
	GetByID(ctx context.Context, id string) (*domain.Order, error)
	UpdateStatus(ctx context.Context, id, status, cancelReason string) (*domain.Order, error)
	List(ctx context.Context, userID, businessID, status string, page, pageSize int32) ([]*domain.Order, int32, error)
}

type orderRepository struct {
	db *gorm.DB
}

func NewOrderRepository(db *gorm.DB) OrderRepository {
	return &orderRepository{db: db}
}

func (r *orderRepository) Create(ctx context.Context, input domain.CreateOrderInput) (*domain.Order, error) {
	o := &domain.Order{
		ID:              uuid.New().String(),
		UserID:          input.UserID,
		BusinessID:      input.BusinessID,
		ShippingAddress: input.ShippingAddress,
		PaymentMethod:   input.PaymentMethod,
		Status:          "pending",
		TotalAmount:     0,
	}
	if err := r.db.WithContext(ctx).Create(o).Error; err != nil {
		return nil, err
	}
	return o, nil
}

func (r *orderRepository) GetByID(ctx context.Context, id string) (*domain.Order, error) {
	var o domain.Order
	err := r.db.WithContext(ctx).Preload("Items").Where("id = ?", id).First(&o).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &o, nil
}

func (r *orderRepository) UpdateStatus(ctx context.Context, id, newStatus, cancelReason string) (*domain.Order, error) {
	updates := map[string]any{"status": newStatus}
	if cancelReason != "" {
		updates["cancel_reason"] = cancelReason
	}

	result := r.db.WithContext(ctx).Model(&domain.Order{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, errors.New("order not found")
	}
	return r.GetByID(ctx, id)
}

func (r *orderRepository) List(ctx context.Context, userID, businessID, statusFilter string, page, pageSize int32) ([]*domain.Order, int32, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	query := r.db.WithContext(ctx).Model(&domain.Order{})
	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}
	if businessID != "" {
		query = query.Where("business_id = ?", businessID)
	}
	if statusFilter != "" {
		query = query.Where("status = ?", statusFilter)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var orders []*domain.Order
	offset := (page - 1) * pageSize
	err := query.Preload("Items").
		Order("created_at DESC").
		Limit(int(pageSize)).
		Offset(int(offset)).
		Find(&orders).Error
	if err != nil {
		return nil, 0, err
	}

	return orders, int32(total), nil
}
