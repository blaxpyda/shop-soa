package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"thugcorp.io/grocery/cart/internal/domain"
)

type CartRepository interface {
	GetCart(ctx context.Context, userID string) (*domain.Cart, error)
	AddItem(ctx context.Context, userID, productID string, quantity int32) error
	RemoveItem(ctx context.Context, userID, productID string) error
	ClearCart(ctx context.Context, userID string) error
}

type cartRepository struct {
	redis *redis.Client
}

func NewCartRepository(redis *redis.Client) CartRepository {
	return &cartRepository{redis: redis}
}

func (r *cartRepository) GetCart(ctx context.Context, userID string) (*domain.Cart, error) {
	key := fmt.Sprintf("cart:%s", userID)

	data, err := r.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		return &domain.Cart{UserID: userID, Items: []domain.CartItem{}}, nil
	}
	if err != nil {
		return nil, err
	}

	var cart domain.Cart
	if err := json.Unmarshal([]byte(data), &cart); err != nil {
		return nil, err
	}
	return &cart, nil
}

func (r *cartRepository) AddItem(ctx context.Context, userID, productID string, quantity int32) error {
	cart, err := r.GetCart(ctx, userID)
	if err != nil {
		return err
	}

	found := false
	for i, item := range cart.Items {
		if item.ProductID == productID {
			cart.Items[i].Quantity += quantity
			found = true
			break
		}
	}

	if !found {
		cart.Items = append(cart.Items, domain.CartItem{
			ProductID: productID,
			Quantity:  quantity,
		})
	}

	return r.saveCart(ctx, cart)
}

func (r *cartRepository) RemoveItem(ctx context.Context, userID, productID string) error {
	cart, err := r.GetCart(ctx, userID)
	if err != nil {
		return err
	}

	newItems := []domain.CartItem{}
	for _, item := range cart.Items {
		if item.ProductID != productID {
			newItems = append(newItems, item)
		}
	}
	cart.Items = newItems

	return r.saveCart(ctx, cart)
}

func (r *cartRepository) ClearCart(ctx context.Context, userID string) error {
	key := fmt.Sprintf("cart:%s", userID)
	return r.redis.Del(ctx, key).Err()
}

func (r *cartRepository) saveCart(ctx context.Context, cart *domain.Cart) error {
	key := fmt.Sprintf("cart:%s", cart.UserID)

	data, err := json.Marshal(cart)
	if err != nil {
		return err
	}

	return r.redis.Set(ctx, key, data, 7*24*time.Hour).Err()
}
