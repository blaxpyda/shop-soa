package internal

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"thugcorp.io/ordering/internal/clients"
	"thugcorp.io/ordering/internal/domain"
)

type OrderingService interface {
	// Cart
	GetCart(ctx context.Context, userID string) (*domain.Cart, error)
	AddItem(ctx context.Context, userID, productID string, quantity int64) (*domain.Cart, error)
	UpdateItem(ctx context.Context, userID, productID string, quantity int64) (*domain.Cart, error)
	RemoveItem(ctx context.Context, userID, productID string) (*domain.Cart, error)
	ClearCart(ctx context.Context, userID string) error

	// Orders
	Checkout(ctx context.Context, input domain.CheckoutInput) (*domain.Order, error)
	GetOrder(ctx context.Context, orderID string) (*domain.Order, error)
	ListOrders(ctx context.Context, filter domain.ListOrdersFilter) ([]*domain.Order, error)
	UpdateOrderStatus(ctx context.Context, orderID string, status domain.OrderStatus) (*domain.Order, error)
}

type orderingService struct {
	repo          OrderingRepository
	catalogClient clients.CatalogClient
}

func NewOrderingService(repo OrderingRepository, catalogClient clients.CatalogClient) OrderingService {
	return &orderingService{repo: repo, catalogClient: catalogClient}
}

// ---- Cart ----

func (s *orderingService) GetCart(ctx context.Context, userID string) (*domain.Cart, error) {
	if userID == "" {
		return nil, errors.New("user_id is required")
	}
	return s.repo.GetCart(ctx, userID)
}

func (s *orderingService) AddItem(ctx context.Context, userID, productID string, quantity int64) (*domain.Cart, error) {
	if userID == "" {
		return nil, errors.New("user_id is required")
	}
	if productID == "" {
		return nil, errors.New("product_id is required")
	}
	if quantity <= 0 {
		return nil, errors.New("quantity must be positive")
	}

	product, err := s.catalogClient.GetProduct(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("failed to look up product: %w", err)
	}
	if !product.Active {
		return nil, errors.New("product is not available")
	}

	// If the item is already in the cart, add to existing quantity.
	cart, err := s.repo.GetCart(ctx, userID)
	if err != nil {
		return nil, err
	}
	for _, existing := range cart.Items {
		if existing.ProductID == productID {
			quantity += existing.Quantity
			break
		}
	}

	item := domain.CartItem{
		UserID:     userID,
		ProductID:  productID,
		BusinessID: product.BusinessID,
		Name:       product.Name,
		UnitPrice:  product.Price,
		Currency:   product.Currency,
		Quantity:   quantity,
	}
	return s.repo.UpsertCartItem(ctx, userID, item)
}

func (s *orderingService) UpdateItem(ctx context.Context, userID, productID string, quantity int64) (*domain.Cart, error) {
	if userID == "" {
		return nil, errors.New("user_id is required")
	}
	if productID == "" {
		return nil, errors.New("product_id is required")
	}
	// quantity == 0 means remove, per proto spec.
	if quantity <= 0 {
		return s.repo.RemoveCartItem(ctx, userID, productID)
	}

	cart, err := s.repo.GetCart(ctx, userID)
	if err != nil {
		return nil, err
	}
	// Find the existing item to keep its cached fields.
	var existing *domain.CartItem
	for i := range cart.Items {
		if cart.Items[i].ProductID == productID {
			existing = &cart.Items[i]
			break
		}
	}
	if existing == nil {
		return nil, fmt.Errorf("product %s is not in the cart", productID)
	}

	existing.Quantity = quantity
	return s.repo.UpsertCartItem(ctx, userID, *existing)
}

func (s *orderingService) RemoveItem(ctx context.Context, userID, productID string) (*domain.Cart, error) {
	if userID == "" {
		return nil, errors.New("user_id is required")
	}
	if productID == "" {
		return nil, errors.New("product_id is required")
	}
	return s.repo.RemoveCartItem(ctx, userID, productID)
}

func (s *orderingService) ClearCart(ctx context.Context, userID string) error {
	if userID == "" {
		return errors.New("user_id is required")
	}
	return s.repo.ClearCart(ctx, userID)
}

// ---- Orders ----

func (s *orderingService) Checkout(ctx context.Context, input domain.CheckoutInput) (*domain.Order, error) {
	if input.UserID == "" {
		return nil, errors.New("user_id is required")
	}

	// Idempotency: return the existing order if this key was already used.
	if input.IdempotencyKey != "" {
		existing, err := s.repo.GetOrderByIdempotencyKey(ctx, input.IdempotencyKey)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			return existing, nil
		}
	}

	cart, err := s.repo.GetCart(ctx, input.UserID)
	if err != nil {
		return nil, err
	}
	if len(cart.Items) == 0 {
		return nil, errors.New("cart is empty")
	}

	// Snapshot cart items → order items.
	var orderItems []domain.OrderItem
	var total int64
	currency := cart.DisplayCurrency()

	for _, ci := range cart.Items {
		lineTotal := ci.UnitPrice * ci.Quantity
		total += lineTotal
		orderItems = append(orderItems, domain.OrderItem{
			ID:          uuid.New().String(),
			ProductID:   ci.ProductID,
			BusinessID:  ci.BusinessID,
			ProductName: ci.Name,
			UnitPrice:   ci.UnitPrice,
			Quantity:    ci.Quantity,
			LineTotal:   lineTotal,
		})
	}

	order := &domain.Order{
		ID:              uuid.New().String(),
		UserID:          input.UserID,
		Status:          domain.OrderStatusPendingPayment,
		Total:           total,
		Currency:        currency,
		DeliveryAddress: input.DeliveryAddress,
		IdempotencyKey:  input.IdempotencyKey,
		Items:           orderItems,
	}

	return s.repo.CreateOrderAndClearCart(ctx, input.UserID, order)
}

func (s *orderingService) GetOrder(ctx context.Context, orderID string) (*domain.Order, error) {
	if orderID == "" {
		return nil, errors.New("order_id is required")
	}
	order, err := s.repo.GetOrderByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, errors.New("order not found")
	}
	return order, nil
}

func (s *orderingService) ListOrders(ctx context.Context, filter domain.ListOrdersFilter) ([]*domain.Order, error) {
	if filter.UserID == "" && filter.BusinessID == "" {
		return nil, errors.New("user_id or business_id filter is required")
	}
	return s.repo.ListOrders(ctx, filter)
}

func (s *orderingService) UpdateOrderStatus(ctx context.Context, orderID string, status domain.OrderStatus) (*domain.Order, error) {
	if orderID == "" {
		return nil, errors.New("order_id is required")
	}
	if status == domain.OrderStatusUnspecified {
		return nil, errors.New("status is required")
	}
	return s.repo.UpdateOrderStatus(ctx, orderID, status)
}
