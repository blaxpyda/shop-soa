package internal

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"thugcorp.io/catalog/internal/domain"
)

func newUUID() string {
	return uuid.New().String()
}

type CatalogRepository interface {
	// Products
	CreateProduct(ctx context.Context, input domain.CreateProductInput) (*domain.Product, error)
	GetProductByID(ctx context.Context, productID string) (*domain.Product, error)
	UpdateProduct(ctx context.Context, productID string, input domain.UpdateProductInput) (*domain.Product, error)
	ListProducts(ctx context.Context, filter domain.ListProductsFilter) ([]*domain.Product, error)

	// Stock
	GetStockItem(ctx context.Context, productID, locationID string) (*domain.StockItem, error)
	GetStockItemsByProductIDs(ctx context.Context, productIDs []string) ([]*domain.StockItem, error)
	AdjustStock(ctx context.Context, input domain.AdjustStockInput) (*domain.StockItem, error)

	// Availability
	CheckAvailability(ctx context.Context, queries []domain.AvailabilityQuery) ([]*domain.AvailabilityResult, error)

	// Reservations (all atomic)
	Reserve(ctx context.Context, input domain.ReserveInput) (*domain.Reservation, []*domain.AvailabilityResult, bool, error)
	GetReservation(ctx context.Context, reservationID string) (*domain.Reservation, error)
	GetReservationByIdempotencyKey(ctx context.Context, key string) (*domain.Reservation, error)
	CommitReservation(ctx context.Context, reservationID string) error
	ReleaseReservation(ctx context.Context, reservationID string) error
}

type catalogRepository struct {
	db *gorm.DB
}

func NewCatalogRepository(db *gorm.DB) CatalogRepository {
	return &catalogRepository{db: db}
}

// ---- Products ----

func (r *catalogRepository) CreateProduct(ctx context.Context, input domain.CreateProductInput) (*domain.Product, error) {
	product := &domain.Product{
		ID:          newUUID(),
		BusinessID:  input.BusinessID,
		Name:        input.Name,
		Description: input.Description,
		Category:    input.Category,
		Price:       input.Price,
		CostPrice:   input.CostPrice,
		Currency:    input.Currency,
		ImageURL:    input.ImageURL,
		Active:      true,
	}
	stock := &domain.StockItem{
		ProductID:  product.ID,
		BusinessID: input.BusinessID,
		OnHand:     input.InitialStock,
		Reserved:   0,
		Version:    newUUID(),
	}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(product).Error; err != nil {
			return err
		}
		return tx.Create(stock).Error
	})
	if err != nil {
		return nil, err
	}
	return product, nil
}

func (r *catalogRepository) GetProductByID(ctx context.Context, productID string) (*domain.Product, error) {
	var product domain.Product
	if err := r.db.WithContext(ctx).Where("id = ?", productID).First(&product).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &product, nil
}

func (r *catalogRepository) UpdateProduct(ctx context.Context, productID string, input domain.UpdateProductInput) (*domain.Product, error) {
	updates := map[string]interface{}{}
	if input.Name != "" {
		updates["name"] = input.Name
	}
	if input.Description != "" {
		updates["description"] = input.Description
	}
	if input.Category != "" {
		updates["category"] = input.Category
	}
	if input.Price > 0 {
		updates["price"] = input.Price
	}
	if input.CostPrice > 0 {
		updates["cost_price"] = input.CostPrice
	}
	if input.Active != nil {
		updates["active"] = *input.Active
	}

	result := r.db.WithContext(ctx).Model(&domain.Product{}).Where("id = ?", productID).Updates(updates)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, errors.New("product not found")
	}
	return r.GetProductByID(ctx, productID)
}

func (r *catalogRepository) ListProducts(ctx context.Context, filter domain.ListProductsFilter) ([]*domain.Product, error) {
	query := r.db.WithContext(ctx).Model(&domain.Product{}).Where("active = ?", true)
	if filter.BusinessID != "" {
		query = query.Where("business_id = ?", filter.BusinessID)
	}
	if filter.Query != "" {
		like := "%" + filter.Query + "%"
		query = query.Where("name LIKE ? OR description LIKE ?", like, like)
	}

	pageSize := filter.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	if filter.PageToken != "" {
		if offset, err := strconv.Atoi(filter.PageToken); err == nil && offset > 0 {
			query = query.Offset(offset)
		}
	}

	var products []*domain.Product
	if err := query.Order("created_at ASC, id ASC").Limit(pageSize).Find(&products).Error; err != nil {
		return nil, err
	}
	return products, nil
}

// ---- Stock ----

func stockQuery(db *gorm.DB, productID, locationID string) *gorm.DB {
	q := db.Where("product_id = ?", productID)
	if locationID != "" {
		q = q.Where("location_id = ?", locationID)
	}
	return q
}

func (r *catalogRepository) GetStockItem(ctx context.Context, productID, locationID string) (*domain.StockItem, error) {
	var item domain.StockItem
	if err := stockQuery(r.db.WithContext(ctx), productID, locationID).
		First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (r *catalogRepository) GetStockItemsByProductIDs(ctx context.Context, productIDs []string) ([]*domain.StockItem, error) {
	var items []*domain.StockItem
	if err := r.db.WithContext(ctx).Where("product_id IN ?", productIDs).Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *catalogRepository) AdjustStock(ctx context.Context, input domain.AdjustStockInput) (*domain.StockItem, error) {
	var result *domain.StockItem

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Idempotency check
		if input.IdempotencyKey != "" {
			var log domain.AdjustmentLog
			if err := tx.Where("idempotency_key = ?", input.IdempotencyKey).First(&log).Error; err == nil {
				var stock domain.StockItem
				if err := stockQuery(tx, input.ProductID, input.LocationID).First(&stock).Error; err != nil {
					return err
				}
				result = &stock
				return nil
			}
		}

		var stock domain.StockItem
		if err := stockQuery(tx, input.ProductID, input.LocationID).First(&stock).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("stock item not found for product %s", input.ProductID)
			}
			return err
		}

		if input.ExpectedVersion != "" && input.ExpectedVersion != stock.Version {
			return fmt.Errorf("version mismatch: expected %q, have %q", input.ExpectedVersion, stock.Version)
		}

		var newOnHand int64
		switch {
		case input.Delta != nil:
			newOnHand = stock.OnHand + *input.Delta
		case input.SetTo != nil:
			newOnHand = *input.SetTo
		default:
			return errors.New("either delta or set_to must be provided")
		}

		if newOnHand < stock.Reserved {
			return fmt.Errorf("on_hand (%d) cannot drop below reserved (%d)", newOnHand, stock.Reserved)
		}

		newVersion := newUUID()
		updateQ := stockQuery(tx.Model(&domain.StockItem{}), input.ProductID, input.LocationID).
			Where("version = ?", stock.Version)
		if err := updateQ.Updates(map[string]interface{}{"on_hand": newOnHand, "version": newVersion}).Error; err != nil {
			return err
		}

		if input.IdempotencyKey != "" {
			entry := domain.AdjustmentLog{IdempotencyKey: input.IdempotencyKey, ProductID: input.ProductID, UnitCost: input.UnitCost}
			if err := tx.Create(&entry).Error; err != nil {
				return err
			}
		}

		stock.OnHand = newOnHand
		stock.Version = newVersion
		result = &stock
		return nil
	})

	return result, err
}

// ---- Availability ----

func (r *catalogRepository) CheckAvailability(ctx context.Context, queries []domain.AvailabilityQuery) ([]*domain.AvailabilityResult, error) {
	productIDs := make([]string, len(queries))
	for i, q := range queries {
		productIDs[i] = q.ProductID
	}

	items, err := r.GetStockItemsByProductIDs(ctx, productIDs)
	if err != nil {
		return nil, err
	}

	stockMap := make(map[string]*domain.StockItem, len(items))
	for _, item := range items {
		stockMap[item.ProductID] = item
	}

	results := make([]*domain.AvailabilityResult, len(queries))
	for i, q := range queries {
		avail := int64(0)
		if item, ok := stockMap[q.ProductID]; ok {
			avail = item.Available()
		}
		results[i] = &domain.AvailabilityResult{
			ProductID:  q.ProductID,
			Available:  avail,
			Sufficient: avail >= q.Quantity,
		}
	}
	return results, nil
}

// ---- Reservations ----

func (r *catalogRepository) Reserve(ctx context.Context, input domain.ReserveInput) (*domain.Reservation, []*domain.AvailabilityResult, bool, error) {
	// Fast-path: idempotent replay
	if input.IdempotencyKey != "" {
		existing, err := r.GetReservationByIdempotencyKey(ctx, input.IdempotencyKey)
		if err != nil {
			return nil, nil, false, err
		}
		if existing != nil {
			return existing, nil, existing.Status == domain.ReservationStatusPending, nil
		}
	}

	var reservation *domain.Reservation
	var shortfalls []*domain.AvailabilityResult
	held := false

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Check stock for every item; collect shortfalls.
		allOK := true
		for _, item := range input.Items {
			var stock domain.StockItem
			if err := stockQuery(tx, item.ProductID, item.LocationID).First(&stock).Error; err != nil {
				return fmt.Errorf("stock not found for product %s: %w", item.ProductID, err)
			}
			if avail := stock.Available(); avail < item.Quantity {
				allOK = false
				shortfalls = append(shortfalls, &domain.AvailabilityResult{
					ProductID:  item.ProductID,
					Available:  avail,
					Sufficient: false,
				})
			}
		}

		if !allOK {
			return nil // insufficient stock; not an error, just held=false
		}

		// Atomically increment reserved for each item.
		for _, item := range input.Items {
			if err := stockQuery(tx.Model(&domain.StockItem{}), item.ProductID, item.LocationID).
				Update("reserved", gorm.Expr("reserved + ?", item.Quantity)).Error; err != nil {
				return err
			}
		}

		ttl := input.TTL
		if ttl <= 0 {
			ttl = 10 * time.Minute
		}

		res := &domain.Reservation{
			ID:             newUUID(),
			OrderID:        input.OrderID,
			IdempotencyKey: input.IdempotencyKey,
			Status:         domain.ReservationStatusPending,
			ExpiresAt:      time.Now().Add(ttl),
		}
		for _, item := range input.Items {
			res.Items = append(res.Items, domain.ReservationItem{
				ID:            newUUID(),
				ReservationID: res.ID,
				ProductID:     item.ProductID,
				LocationID:    item.LocationID,
				Quantity:      item.Quantity,
			})
		}

		if err := tx.Create(res).Error; err != nil {
			return err
		}

		reservation = res
		held = true
		return nil
	})

	return reservation, shortfalls, held, err
}

func (r *catalogRepository) GetReservation(ctx context.Context, reservationID string) (*domain.Reservation, error) {
	var res domain.Reservation
	if err := r.db.WithContext(ctx).Preload("Items").
		Where("id = ?", reservationID).First(&res).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &res, nil
}

func (r *catalogRepository) GetReservationByIdempotencyKey(ctx context.Context, key string) (*domain.Reservation, error) {
	var res domain.Reservation
	if err := r.db.WithContext(ctx).Preload("Items").
		Where("idempotency_key = ?", key).First(&res).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &res, nil
}

func (r *catalogRepository) CommitReservation(ctx context.Context, reservationID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var res domain.Reservation
		if err := tx.Preload("Items").
			Where("id = ? AND status = ?", reservationID, domain.ReservationStatusPending).
			First(&res).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("reservation %s not found or not pending", reservationID)
			}
			return err
		}

		for _, item := range res.Items {
			if err := stockQuery(tx.Model(&domain.StockItem{}), item.ProductID, item.LocationID).
				Updates(map[string]interface{}{
					"on_hand":  gorm.Expr("on_hand - ?", item.Quantity),
					"reserved": gorm.Expr("reserved - ?", item.Quantity),
					"version":  newUUID(),
				}).Error; err != nil {
				return err
			}
		}

		return tx.Model(&domain.Reservation{}).
			Where("id = ?", reservationID).
			Update("status", domain.ReservationStatusCommitted).Error
	})
}

func (r *catalogRepository) ReleaseReservation(ctx context.Context, reservationID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var res domain.Reservation
		if err := tx.Preload("Items").
			Where("id = ? AND status = ?", reservationID, domain.ReservationStatusPending).
			First(&res).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("reservation %s not found or not pending", reservationID)
			}
			return err
		}

		for _, item := range res.Items {
			if err := stockQuery(tx.Model(&domain.StockItem{}), item.ProductID, item.LocationID).
				Update("reserved", gorm.Expr("reserved - ?", item.Quantity)).Error; err != nil {
				return err
			}
		}

		return tx.Model(&domain.Reservation{}).
			Where("id = ?", reservationID).
			Update("status", domain.ReservationStatusReleased).Error
	})
}
