package internal

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"thugcorp.io/ordering/internal/domain"
)

func newUUID() string {
	return uuid.New().String()
}

type OrderingRepository interface {
	// Cart
	GetCart(ctx context.Context, userID string) (*domain.Cart, error)
	UpsertCartItem(ctx context.Context, userID string, item domain.CartItem) (*domain.Cart, error)
	RemoveCartItem(ctx context.Context, userID, productID string) (*domain.Cart, error)
	ClearCart(ctx context.Context, userID string) error

	// Orders
	GetOrderByID(ctx context.Context, orderID string) (*domain.Order, error)
	GetOrderByIdempotencyKey(ctx context.Context, key string) (*domain.Order, error)
	ListOrders(ctx context.Context, filter domain.ListOrdersFilter) ([]*domain.Order, error)
	UpdateOrderStatus(ctx context.Context, orderID string, status domain.OrderStatus) (*domain.Order, error)

	// Checkout: atomically create order + clear cart.
	CreateOrderAndClearCart(ctx context.Context, userID string, order *domain.Order) (*domain.Order, error)
}

type orderingRepository struct {
	db *gorm.DB
}

func NewOrderingRepository(db *gorm.DB) OrderingRepository {
	return &orderingRepository{db: db}
}

// ---- Cart ----

func (r *orderingRepository) GetCart(ctx context.Context, userID string) (*domain.Cart, error) {
	var cart domain.Cart
	err := r.db.WithContext(ctx).Preload("Items").Where("user_id = ?", userID).First(&cart).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Return an empty cart — it doesn't exist in the DB yet.
			return &domain.Cart{UserID: userID, Items: []domain.CartItem{}}, nil
		}
		return nil, err
	}
	return &cart, nil
}

func (r *orderingRepository) UpsertCartItem(ctx context.Context, userID string, item domain.CartItem) (*domain.Cart, error) {
	item.UserID = userID

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Ensure the cart row exists; seed currency from the item on first create.
		cart := domain.Cart{UserID: userID}
		if err := tx.Where(domain.Cart{UserID: userID}).
			Attrs(domain.Cart{Currency: item.Currency}).
			FirstOrCreate(&cart).Error; err != nil {
			return err
		}
		return tx.Save(&item).Error
	})
	if err != nil {
		return nil, err
	}
	return r.GetCart(ctx, userID)
}

func (r *orderingRepository) RemoveCartItem(ctx context.Context, userID, productID string) (*domain.Cart, error) {
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND product_id = ?", userID, productID).
		Delete(&domain.CartItem{}).Error; err != nil {
		return nil, err
	}
	return r.GetCart(ctx, userID)
}

func (r *orderingRepository) ClearCart(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&domain.CartItem{}).Error; err != nil {
			return err
		}
		return tx.Where("user_id = ?", userID).Delete(&domain.Cart{}).Error
	})
}

// ---- Orders ----

func (r *orderingRepository) GetOrderByID(ctx context.Context, orderID string) (*domain.Order, error) {
	var order domain.Order
	if err := r.db.WithContext(ctx).Preload("Items").Where("id = ?", orderID).First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &order, nil
}

func (r *orderingRepository) GetOrderByIdempotencyKey(ctx context.Context, key string) (*domain.Order, error) {
	var order domain.Order
	if err := r.db.WithContext(ctx).Preload("Items").Where("idempotency_key = ?", key).First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &order, nil
}

func (r *orderingRepository) ListOrders(ctx context.Context, filter domain.ListOrdersFilter) ([]*domain.Order, error) {
	query := r.db.WithContext(ctx).Model(&domain.Order{})

	if !filter.AllOrders {
		switch {
		case filter.UserID != "":
			query = query.Where("user_id = ?", filter.UserID)
		case filter.BusinessID != "":
			subquery := r.db.Model(&domain.OrderItem{}).
				Select("order_id").
				Where("business_id = ?", filter.BusinessID)
			query = query.Where("id IN (?)", subquery)
		}
	}

	if filter.Status != "" && filter.Status != domain.OrderStatusUnspecified {
		query = query.Where("status = ?", filter.Status)
	}

	pageSize := filter.PageSize
	maxPageSize := 100
	if filter.AllOrders {
		maxPageSize = 1000
	}
	if pageSize <= 0 || pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	if filter.PageToken != "" {
		if offset, err := strconv.Atoi(filter.PageToken); err == nil && offset > 0 {
			query = query.Offset(offset)
		}
	}

	var orders []*domain.Order
	if err := query.Order("created_at DESC").Limit(pageSize).Preload("Items").Find(&orders).Error; err != nil {
		return nil, err
	}
	return orders, nil
}

func (r *orderingRepository) UpdateOrderStatus(ctx context.Context, orderID string, status domain.OrderStatus) (*domain.Order, error) {
	result := r.db.WithContext(ctx).Model(&domain.Order{}).
		Where("id = ?", orderID).
		Update("status", status)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, fmt.Errorf("order %s not found", orderID)
	}
	return r.GetOrderByID(ctx, orderID)
}

// CreateOrderAndClearCart persists the order and wipes the user's cart in one
// transaction so checkout is atomic.
func (r *orderingRepository) CreateOrderAndClearCart(ctx context.Context, userID string, order *domain.Order) (*domain.Order, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(order).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).Delete(&domain.CartItem{}).Error; err != nil {
			return err
		}
		return tx.Where("user_id = ?", userID).Delete(&domain.Cart{}).Error
	})
	if err != nil {
		return nil, err
	}
	return r.GetOrderByID(ctx, order.ID)
}

