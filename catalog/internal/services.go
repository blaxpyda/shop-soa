package internal

import (
	"context"
	"errors"
	"time"

	"thugcorp.io/catalog/internal/domain"
)

type CatalogService interface {
	CreateProduct(ctx context.Context, businessID string, input domain.CreateProductInput) (*domain.Product, error)
	GetProduct(ctx context.Context, productID string) (*domain.Product, error)
	UpdateProduct(ctx context.Context, productID, callerBusinessID string, input domain.UpdateProductInput) (*domain.Product, error)
	ListProducts(ctx context.Context, filter domain.ListProductsFilter) ([]*domain.Product, error)

	CheckAvailability(ctx context.Context, queries []domain.AvailabilityQuery) ([]*domain.AvailabilityResult, error)
	AdjustStock(ctx context.Context, input domain.AdjustStockInput) (*domain.StockItem, error)

	Reserve(ctx context.Context, input domain.ReserveInput) (*domain.Reservation, []*domain.AvailabilityResult, bool, error)
	CommitReservation(ctx context.Context, reservationID string) error
	ReleaseReservation(ctx context.Context, reservationID string) error
}

type catalogService struct {
	repo CatalogRepository
}

func NewCatalogService(repo CatalogRepository) CatalogService {
	return &catalogService{repo: repo}
}

// ---- Products ----

func (s *catalogService) CreateProduct(ctx context.Context, businessID string, input domain.CreateProductInput) (*domain.Product, error) {
	if businessID == "" {
		return nil, errors.New("business_id is required")
	}
	if input.Name == "" {
		return nil, errors.New("name is required")
	}
	if input.Price <= 0 {
		return nil, errors.New("price must be positive")
	}
	if input.Currency == "" {
		return nil, errors.New("currency is required")
	}
	if input.InitialStock < 0 {
		return nil, errors.New("initial_stock cannot be negative")
	}
	input.BusinessID = businessID
	return s.repo.CreateProduct(ctx, input)
}

func (s *catalogService) GetProduct(ctx context.Context, productID string) (*domain.Product, error) {
	if productID == "" {
		return nil, errors.New("product_id is required")
	}
	product, err := s.repo.GetProductByID(ctx, productID)
	if err != nil {
		return nil, err
	}
	if product == nil {
		return nil, errors.New("product not found")
	}
	return product, nil
}

func (s *catalogService) UpdateProduct(ctx context.Context, productID, callerBusinessID string, input domain.UpdateProductInput) (*domain.Product, error) {
	product, err := s.repo.GetProductByID(ctx, productID)
	if err != nil {
		return nil, err
	}
	if product == nil {
		return nil, errors.New("product not found")
	}
	if callerBusinessID != "" && product.BusinessID != callerBusinessID {
		return nil, errors.New("unauthorized: product does not belong to your business")
	}
	return s.repo.UpdateProduct(ctx, productID, input)
}

func (s *catalogService) ListProducts(ctx context.Context, filter domain.ListProductsFilter) ([]*domain.Product, error) {
	return s.repo.ListProducts(ctx, filter)
}

// ---- Inventory ----

func (s *catalogService) CheckAvailability(ctx context.Context, queries []domain.AvailabilityQuery) ([]*domain.AvailabilityResult, error) {
	if len(queries) == 0 {
		return nil, errors.New("items cannot be empty")
	}
	return s.repo.CheckAvailability(ctx, queries)
}

func (s *catalogService) AdjustStock(ctx context.Context, input domain.AdjustStockInput) (*domain.StockItem, error) {
	if input.ProductID == "" {
		return nil, errors.New("product_id is required")
	}
	if input.Delta == nil && input.SetTo == nil {
		return nil, errors.New("either delta or set_to must be specified")
	}
	return s.repo.AdjustStock(ctx, input)
}

// ---- Checkout saga ----

func (s *catalogService) Reserve(ctx context.Context, input domain.ReserveInput) (*domain.Reservation, []*domain.AvailabilityResult, bool, error) {
	if input.OrderID == "" {
		return nil, nil, false, errors.New("order_id is required")
	}
	if len(input.Items) == 0 {
		return nil, nil, false, errors.New("items cannot be empty")
	}
	if input.TTL <= 0 {
		input.TTL = 10 * time.Minute
	}
	return s.repo.Reserve(ctx, input)
}

func (s *catalogService) CommitReservation(ctx context.Context, reservationID string) error {
	if reservationID == "" {
		return errors.New("reservation_id is required")
	}
	return s.repo.CommitReservation(ctx, reservationID)
}

func (s *catalogService) ReleaseReservation(ctx context.Context, reservationID string) error {
	if reservationID == "" {
		return errors.New("reservation_id is required")
	}
	return s.repo.ReleaseReservation(ctx, reservationID)
}
