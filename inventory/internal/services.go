package internal

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"thugcorp.io/grocery/inventory/internal/domain"
)

type InventoryService interface {
	CheckAvailability(ctx context.Context, queries []AvailabilityQuery) ([]AvailabilityResult, error)
	GetStock(ctx context.Context, businessID, productID, locationID string) (*domain.StockItem, error)
	Reserve(ctx context.Context, input ReserveInput) (*ReserveResult, error)
	CommitReservation(ctx context.Context, reservationID, idempotencyKey string) error
	ReleaseReservation(ctx context.Context, reservationID, reason string) error
	AdjustStock(ctx context.Context, input AdjustStockInput) (*domain.StockItem, error)
	ListStock(ctx context.Context, businessID, locationID, stateFilter string, pageSize int, cursor string) ([]*domain.StockItem, string, error)
}

// ---- Input / result types ----

type AvailabilityQuery struct {
	BusinessID string
	ProductID  string
	LocationID string
	Quantity   int64
}

type AvailabilityResult struct {
	ProductID  string
	LocationID string
	Available  int64
	Sufficient bool
	State      string
}

type ReserveInput struct {
	OrderID        string
	Items          []ReserveLineInput
	TTL            time.Duration
	IdempotencyKey string
}

type ReserveLineInput struct {
	BusinessID string
	ProductID  string
	LocationID string
	Quantity   int64
}

type ReserveResult struct {
	ReservationID string
	Status        string // RESERVATION_STATUS_HELD | RESERVATION_STATUS_FAILED
	ExpiresAt     time.Time
	Shortfalls    []domain.ShortfallResult
}

type AdjustStockInput struct {
	BusinessID      string
	ProductID       string
	LocationID      string
	Delta           *int64 // relative adjustment
	SetTo           *int64 // absolute value
	ExpectedVersion string
	IdempotencyKey  string
}

// ---- Implementation ----

type inventoryService struct {
	repo InventoryRepository
}

func NewInventoryService(repo InventoryRepository) InventoryService {
	return &inventoryService{repo: repo}
}

func (s *inventoryService) CheckAvailability(ctx context.Context, queries []AvailabilityQuery) ([]AvailabilityResult, error) {
	results := make([]AvailabilityResult, 0, len(queries))
	for _, q := range queries {
		item, err := s.repo.GetStock(ctx, q.BusinessID, q.ProductID, q.LocationID)
		if err != nil {
			return nil, err
		}
		if item == nil {
			results = append(results, AvailabilityResult{
				ProductID:  q.ProductID,
				LocationID: q.LocationID,
				Available:  0,
				Sufficient: false,
				State:      "STOCK_STATE_OUT_OF_STOCK",
			})
			continue
		}
		avail := item.Available()
		results = append(results, AvailabilityResult{
			ProductID:  q.ProductID,
			LocationID: q.LocationID,
			Available:  avail,
			Sufficient: avail >= q.Quantity,
			State:      item.State,
		})
	}
	return results, nil
}

func (s *inventoryService) GetStock(ctx context.Context, businessID, productID, locationID string) (*domain.StockItem, error) {
	item, err := s.repo.GetStock(ctx, businessID, productID, locationID)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, errors.New("stock item not found")
	}
	return item, nil
}

func (s *inventoryService) Reserve(ctx context.Context, input ReserveInput) (*ReserveResult, error) {
	// Idempotency: return the same result for a duplicate request.
	if input.IdempotencyKey != "" {
		existing, err := s.repo.GetReservationByIdempotencyKey(ctx, input.IdempotencyKey)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			return &ReserveResult{
				ReservationID: existing.ID,
				Status:        existing.Status,
				ExpiresAt:     existing.ExpiresAt,
			}, nil
		}
	}

	ttl := input.TTL
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}

	items := make([]domain.ReservationItem, 0, len(input.Items))
	for _, line := range input.Items {
		items = append(items, domain.ReservationItem{
			BusinessID: line.BusinessID,
			ProductID:  line.ProductID,
			LocationID: line.LocationID,
			Quantity:   line.Quantity,
		})
	}

	reservation := &domain.Reservation{
		ID:             uuid.New().String(),
		OrderID:        input.OrderID,
		Status:         "RESERVATION_STATUS_HELD",
		ExpiresAt:      time.Now().Add(ttl),
		IdempotencyKey: input.IdempotencyKey,
		Items:          items,
	}

	shortfalls, err := s.repo.ReserveAtomically(ctx, reservation)
	if err != nil {
		return nil, err
	}

	if len(shortfalls) > 0 {
		return &ReserveResult{
			ReservationID: reservation.ID,
			Status:        "RESERVATION_STATUS_FAILED",
			Shortfalls:    shortfalls,
		}, nil
	}

	return &ReserveResult{
		ReservationID: reservation.ID,
		Status:        "RESERVATION_STATUS_HELD",
		ExpiresAt:     reservation.ExpiresAt,
	}, nil
}

func (s *inventoryService) CommitReservation(ctx context.Context, reservationID, _ string) error {
	return s.repo.CommitAtomically(ctx, reservationID)
}

func (s *inventoryService) ReleaseReservation(ctx context.Context, reservationID, _ string) error {
	return s.repo.ReleaseAtomically(ctx, reservationID)
}

func (s *inventoryService) AdjustStock(ctx context.Context, input AdjustStockInput) (*domain.StockItem, error) {
	if input.Delta == nil && input.SetTo == nil {
		return nil, errors.New("either delta or set_to must be provided")
	}
	if input.Delta != nil {
		return s.repo.AdjustStock(ctx, input.BusinessID, input.ProductID, input.LocationID, *input.Delta, input.ExpectedVersion, input.IdempotencyKey)
	}
	return s.repo.SetStock(ctx, input.BusinessID, input.ProductID, input.LocationID, *input.SetTo, input.ExpectedVersion, input.IdempotencyKey)
}

func (s *inventoryService) ListStock(ctx context.Context, businessID, locationID, stateFilter string, pageSize int, cursor string) ([]*domain.StockItem, string, error) {
	return s.repo.ListStock(ctx, businessID, locationID, stateFilter, pageSize, cursor)
}
