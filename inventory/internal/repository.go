package internal

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"thugcorp.io/grocery/inventory/internal/domain"
)

type InventoryRepository interface {
	// Stock reads
	GetStock(ctx context.Context, businessID, productID, locationID string) (*domain.StockItem, error)
	ListStock(ctx context.Context, businessID, locationID, stateFilter string, pageSize int, cursor string) ([]*domain.StockItem, string, error)

	// Stock writes (optimistic concurrency via expectedVersion)
	AdjustStock(ctx context.Context, businessID, productID, locationID string, delta int64, expectedVersion, idempotencyKey string) (*domain.StockItem, error)
	SetStock(ctx context.Context, businessID, productID, locationID string, value int64, expectedVersion, idempotencyKey string) (*domain.StockItem, error)

	// Reservation lifecycle (all atomic)
	ReserveAtomically(ctx context.Context, r *domain.Reservation) (shortfalls []domain.ShortfallResult, err error)
	GetReservation(ctx context.Context, id string) (*domain.Reservation, error)
	GetReservationByIdempotencyKey(ctx context.Context, key string) (*domain.Reservation, error)
	CommitAtomically(ctx context.Context, id string) error
	ReleaseAtomically(ctx context.Context, id string) error
}

type inventoryRepository struct {
	db *gorm.DB
}

func NewInventoryRepository(db *gorm.DB) InventoryRepository {
	return &inventoryRepository{db: db}
}

// ---- Stock reads ----

func (r *inventoryRepository) GetStock(ctx context.Context, businessID, productID, locationID string) (*domain.StockItem, error) {
	var item domain.StockItem
	err := r.db.WithContext(ctx).
		Where("business_id = ? AND product_id = ? AND location_id = ?", businessID, productID, locationID).
		First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (r *inventoryRepository) ListStock(ctx context.Context, businessID, locationID, stateFilter string, pageSize int, cursor string) ([]*domain.StockItem, string, error) {
	if pageSize <= 0 {
		pageSize = 20
	}
	q := r.db.WithContext(ctx).Where("business_id = ?", businessID).Order("product_id ASC")
	if locationID != "" {
		q = q.Where("location_id = ?", locationID)
	}
	if stateFilter != "" && stateFilter != "STOCK_STATE_UNSPECIFIED" {
		q = q.Where("state = ?", stateFilter)
	}
	if cursor != "" {
		q = q.Where("product_id > ?", cursor)
	}

	var items []*domain.StockItem
	if err := q.Limit(pageSize + 1).Find(&items).Error; err != nil {
		return nil, "", err
	}

	var nextCursor string
	if len(items) > pageSize {
		nextCursor = items[pageSize-1].ProductID
		items = items[:pageSize]
	}
	return items, nextCursor, nil
}

// ---- Stock writes ----

func (r *inventoryRepository) AdjustStock(ctx context.Context, businessID, productID, locationID string, delta int64, expectedVersion, idempotencyKey string) (*domain.StockItem, error) {
	var result *domain.StockItem
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		item, err := lockStock(tx, businessID, productID, locationID)
		if err != nil {
			return err
		}

		// Upsert: create with delta as initial on_hand if not found.
		if item == nil {
			item = &domain.StockItem{
				BusinessID: businessID,
				ProductID:  productID,
				LocationID: locationID,
				OnHand:     max64(0, delta),
				Version:    uuid.New().String(),
			}
			item.State = domain.ComputeState(item)
			result = item
			return tx.Create(item).Error
		}

		if expectedVersion != "" && item.Version != expectedVersion {
			return errors.New("version conflict: stock was modified concurrently")
		}

		item.OnHand = max64(0, item.OnHand+delta)
		item.Version = uuid.New().String()
		item.State = domain.ComputeState(item)
		result = item
		return tx.Save(item).Error
	})
	return result, err
}

func (r *inventoryRepository) SetStock(ctx context.Context, businessID, productID, locationID string, value int64, expectedVersion, idempotencyKey string) (*domain.StockItem, error) {
	var result *domain.StockItem
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		item, err := lockStock(tx, businessID, productID, locationID)
		if err != nil {
			return err
		}

		if item == nil {
			item = &domain.StockItem{
				BusinessID: businessID,
				ProductID:  productID,
				LocationID: locationID,
				OnHand:     max64(0, value),
				Version:    uuid.New().String(),
			}
			item.State = domain.ComputeState(item)
			result = item
			return tx.Create(item).Error
		}

		if expectedVersion != "" && item.Version != expectedVersion {
			return errors.New("version conflict: stock was modified concurrently")
		}

		item.OnHand = max64(0, value)
		item.Version = uuid.New().String()
		item.State = domain.ComputeState(item)
		result = item
		return tx.Save(item).Error
	})
	return result, err
}

// ---- Reservation lifecycle ----

// ReserveAtomically checks all lines and holds them in one transaction.
// Returns shortfalls if any line has insufficient stock (no partial holds).
func (r *inventoryRepository) ReserveAtomically(ctx context.Context, reservation *domain.Reservation) ([]domain.ShortfallResult, error) {
	var shortfalls []domain.ShortfallResult

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Lock all stock rows upfront in a consistent order to avoid deadlocks.
		for _, line := range reservation.Items {
			item, err := lockStock(tx, line.BusinessID, line.ProductID, line.LocationID)
			if err != nil {
				return err
			}
			if item == nil || item.Available() < line.Quantity {
				avail := int64(0)
				state := "STOCK_STATE_OUT_OF_STOCK"
				if item != nil {
					avail = item.Available()
					state = item.State
				}
				shortfalls = append(shortfalls, domain.ShortfallResult{
					ProductID:  line.ProductID,
					LocationID: line.LocationID,
					Available:  avail,
					State:      state,
				})
			}
		}

		if len(shortfalls) > 0 {
			return errInsufficientStock
		}

		// All lines are available — increment reserved.
		for _, line := range reservation.Items {
			if err := tx.Model(&domain.StockItem{}).
				Where("business_id = ? AND product_id = ? AND location_id = ?",
					line.BusinessID, line.ProductID, line.LocationID).
				Updates(map[string]interface{}{
					"reserved": gorm.Expr("reserved + ?", line.Quantity),
					"version":  uuid.New().String(),
				}).Error; err != nil {
				return err
			}
		}

		// Assign IDs to reservation items.
		for i := range reservation.Items {
			reservation.Items[i].ID = uuid.New().String()
		}

		return tx.Create(reservation).Error
	})

	if errors.Is(err, errInsufficientStock) {
		return shortfalls, nil
	}
	return nil, err
}

// CommitAtomically converts a held reservation into a permanent on_hand deduction.
func (r *inventoryRepository) CommitAtomically(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		reservation, err := getReservationWithItems(tx, id)
		if err != nil {
			return err
		}
		if reservation.Status != "RESERVATION_STATUS_HELD" {
			return errors.New("reservation is not in HELD state")
		}

		for _, line := range reservation.Items {
			if err := tx.Model(&domain.StockItem{}).
				Where("business_id = ? AND product_id = ? AND location_id = ?",
					line.BusinessID, line.ProductID, line.LocationID).
				Updates(map[string]interface{}{
					"on_hand":  gorm.Expr("on_hand - ?", line.Quantity),
					"reserved": gorm.Expr("reserved - ?", line.Quantity),
					"version":  uuid.New().String(),
				}).Error; err != nil {
				return err
			}
			// Recompute state after deduction.
			if err := recomputeState(tx, line.BusinessID, line.ProductID, line.LocationID); err != nil {
				return err
			}
		}

		return tx.Model(&domain.Reservation{}).
			Where("id = ?", id).
			Update("status", "RESERVATION_STATUS_COMMITTED").Error
	})
}

// ReleaseAtomically frees held stock back to available.
func (r *inventoryRepository) ReleaseAtomically(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		reservation, err := getReservationWithItems(tx, id)
		if err != nil {
			return err
		}
		if reservation.Status != "RESERVATION_STATUS_HELD" {
			return errors.New("reservation is not in HELD state")
		}

		for _, line := range reservation.Items {
			if err := tx.Model(&domain.StockItem{}).
				Where("business_id = ? AND product_id = ? AND location_id = ?",
					line.BusinessID, line.ProductID, line.LocationID).
				Updates(map[string]interface{}{
					"reserved": gorm.Expr("reserved - ?", line.Quantity),
					"version":  uuid.New().String(),
				}).Error; err != nil {
				return err
			}
			if err := recomputeState(tx, line.BusinessID, line.ProductID, line.LocationID); err != nil {
				return err
			}
		}

		return tx.Model(&domain.Reservation{}).
			Where("id = ?", id).
			Update("status", "RESERVATION_STATUS_RELEASED").Error
	})
}

func (r *inventoryRepository) GetReservation(ctx context.Context, id string) (*domain.Reservation, error) {
	return getReservationWithItems(r.db.WithContext(ctx), id)
}

func (r *inventoryRepository) GetReservationByIdempotencyKey(ctx context.Context, key string) (*domain.Reservation, error) {
	var reservation domain.Reservation
	err := r.db.WithContext(ctx).Preload("Items").
		Where("idempotency_key = ?", key).First(&reservation).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &reservation, nil
}

// ---- Helpers ----

var errInsufficientStock = errors.New("insufficient stock")

func lockStock(tx *gorm.DB, businessID, productID, locationID string) (*domain.StockItem, error) {
	var item domain.StockItem
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("business_id = ? AND product_id = ? AND location_id = ?", businessID, productID, locationID).
		First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &item, err
}

func getReservationWithItems(db *gorm.DB, id string) (*domain.Reservation, error) {
	var r domain.Reservation
	err := db.Preload("Items").Where("id = ?", id).First(&r).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("reservation not found")
		}
		return nil, err
	}
	return &r, nil
}

func recomputeState(tx *gorm.DB, businessID, productID, locationID string) error {
	var item domain.StockItem
	if err := tx.Where("business_id = ? AND product_id = ? AND location_id = ?",
		businessID, productID, locationID).First(&item).Error; err != nil {
		return err
	}
	newState := domain.ComputeState(&item)
	if newState == item.State {
		return nil
	}
	return tx.Model(&domain.StockItem{}).
		Where("business_id = ? AND product_id = ? AND location_id = ?", businessID, productID, locationID).
		Update("state", newState).Error
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
