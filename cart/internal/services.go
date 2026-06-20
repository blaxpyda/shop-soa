package internal

import (
	"context"
	"errors"

	"thugcorp.io/grocery/cart/internal/domain"
)

type CartService interface {
	GetCart(ctx context.Context, userID string) (*domain.Cart, error)
	AddToCart(ctx context.Context, userID, productID string, quantity int32) (*domain.Cart, error)
	RemoveFromCart(ctx context.Context, userID, productID string) (*domain.Cart, error)
	ClearCart(ctx context.Context, userID string) error
}

type cartService struct {
	cartRepository CartRepository
}

func NewCartService(cartRepository CartRepository) *cartService {
	return &cartService{cartRepository: cartRepository}
}

func (s *cartService) GetCart(ctx context.Context, userID string) (*domain.Cart, error) {
	return s.cartRepository.GetCart(ctx, userID)
}

func (s *cartService) AddToCart(ctx context.Context, userID, productID string, quantity int32) (*domain.Cart, error) {
	if quantity <= 0 {
		return nil, errors.New("quantity must be greater than zero")
	}

	err := s.cartRepository.AddItem(ctx, userID, productID, quantity)
	if err != nil {
		return nil, err
	}
	return s.GetCart(ctx, userID)
}

func (s *cartService) RemoveFromCart(ctx context.Context, userID, productID string) (*domain.Cart, error) {
	err := s.cartRepository.RemoveItem(ctx, userID, productID)
	if err != nil {
		return nil, err
	}
	return s.GetCart(ctx, userID)
}

func (s *cartService) ClearCart(ctx context.Context, userID string) error {
	return s.cartRepository.ClearCart(ctx, userID)
}