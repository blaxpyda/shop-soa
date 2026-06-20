package internal

import (
	"context"
	"errors"

	"thugcorp.io/grocery/order/internal/domain"
)

type OrderService interface {
	CreateOrder(ctx context.Context, input domain.CreateOrderInput) (*domain.Order, error)
	GetOrder(ctx context.Context, callerID, callerRole, orderID string) (*domain.Order, error)
	UpdateOrderStatus(ctx context.Context, callerRole, orderID, newStatus string) (*domain.Order, error)
	ListOrders(ctx context.Context, userID, businessID, status string, page, pageSize int32) ([]*domain.Order, int32, error)
	CancelOrder(ctx context.Context, callerID, callerRole, orderID, reason string) (*domain.Order, error)
	CompleteOrder(ctx context.Context, callerID, callerRole, orderID string) (*domain.Order, error)
}

type orderService struct {
	repo OrderRepository
}

func NewOrderService(repo OrderRepository) OrderService {
	return &orderService{repo: repo}
}

func (s *orderService) CreateOrder(ctx context.Context, input domain.CreateOrderInput) (*domain.Order, error) {
	if input.UserID == "" {
		return nil, errors.New("user ID is required")
	}
	if input.BusinessID == "" {
		return nil, errors.New("business ID is required")
	}
	return s.repo.Create(ctx, input)
}

func (s *orderService) GetOrder(ctx context.Context, callerID, callerRole, orderID string) (*domain.Order, error) {
	o, err := s.repo.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, errors.New("order not found")
	}
	if !isOwnerOrAdmin(callerID, callerRole, o.UserID) {
		return nil, errors.New("forbidden: you do not have access to this order")
	}
	return o, nil
}

func (s *orderService) UpdateOrderStatus(ctx context.Context, callerRole, orderID, newStatus string) (*domain.Order, error) {
	if callerRole != "admin" && callerRole != "super-admin" {
		return nil, errors.New("forbidden: only admins can update order status")
	}
	if !isValidStatus(newStatus) {
		return nil, errors.New("invalid order status")
	}
	o, err := s.repo.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, errors.New("order not found")
	}
	return s.repo.UpdateStatus(ctx, orderID, newStatus, "")
}

func (s *orderService) ListOrders(ctx context.Context, userID, businessID, status string, page, pageSize int32) ([]*domain.Order, int32, error) {
	return s.repo.List(ctx, userID, businessID, status, page, pageSize)
}

func (s *orderService) CancelOrder(ctx context.Context, callerID, callerRole, orderID, reason string) (*domain.Order, error) {
	o, err := s.repo.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, errors.New("order not found")
	}
	if !isOwnerOrAdmin(callerID, callerRole, o.UserID) {
		return nil, errors.New("forbidden: you cannot cancel this order")
	}
	if o.Status == "cancelled" {
		return nil, errors.New("order is already cancelled")
	}
	if o.Status == "delivered" {
		return nil, errors.New("delivered orders cannot be cancelled")
	}
	return s.repo.UpdateStatus(ctx, orderID, "cancelled", reason)
}

func (s *orderService) CompleteOrder(ctx context.Context, callerID, callerRole, orderID string) (*domain.Order, error) {
	o, err := s.repo.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, errors.New("order not found")
	}

	if !isOwnerOrAdmin(callerID, callerRole, o.UserID) {
		return nil, errors.New("forbidden: you cannot complete this order")
	}

	if o.Status != "shipped" {
		return nil, errors.New("only shipped orders can be marked as completed")
	}
	
	return s.repo.UpdateStatus(ctx, orderID, "delivered", "")
}

func isOwnerOrAdmin(callerID, callerRole, ownerID string) bool {
	return callerID == ownerID || callerRole == "admin" || callerRole == "super-admin"
}

func isValidStatus(s string) bool {
	switch s {
	case "pending", "confirmed", "processing", "shipped", "delivered", "cancelled":
		return true
	}
	return false
}
