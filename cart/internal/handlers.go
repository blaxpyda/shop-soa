package internal

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"thugcorp.io/grocery/cart/internal/middleware"
	"thugcorp.io/grocery/cart/internal/domain"
	pb "thugcorp.io/grocery/cart/proto"
)

type cartHandler struct {
	pb.UnimplementedCartServiceServer
	cartService CartService
}

func NewCartHandler(cartService CartService) *cartHandler {
	return &cartHandler{cartService: cartService}
}

func (h *cartHandler) GetCart(ctx context.Context, req *pb.GetCartRequest) (*pb.Cart, error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	domainCart, err := h.cartService.GetCart(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get cart: %v", err)
	}

	return h.mapToProtoCart(domainCart), nil
}

func (h *cartHandler) AddToCart(ctx context.Context, req *pb.AddToCartRequest) (*pb.Cart, error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	domainCart, err := h.cartService.AddToCart(ctx, userID, req.ProductId, req.Quantity)
	if err != nil {
		if err.Error() == "quantity must be greater than zero" {
			return nil, status.Errorf(codes.InvalidArgument, "invalid quantity: %v", err)
		}
		return nil, status.Errorf(codes.Internal, "failed to add to cart: %v", err)
	}

	return h.mapToProtoCart(domainCart), nil
}

func (h *cartHandler) RemoveFromCart(ctx context.Context, req *pb.RemoveFromCartRequest) (*pb.Cart, error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	domainCart, err := h.cartService.RemoveFromCart(ctx, userID, req.ProductId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to remove from cart: %v", err)
	}

	return h.mapToProtoCart(domainCart), nil
}

func (h *cartHandler) ClearCart(ctx context.Context, req *pb.ClearCartRequest) (*pb.EmptyResponse, error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	err = h.cartService.ClearCart(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to clear cart: %v", err)
	}

	return &pb.EmptyResponse{}, nil
}

func getUserIDFromContext(ctx context.Context) (string, error) {
	userID, ok := ctx.Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		return "", status.Errorf(codes.Unauthenticated, "user ID not found in context")
	}
	return userID, nil
}

func (h *cartHandler) mapToProtoCart(cart *domain.Cart) *pb.Cart {
	pbCart := &pb.Cart{
		UserId: cart.UserID,
	}

	for _, item := range cart.Items {
		pbCart.Items = append(pbCart.Items, &pb.CartItem{
			ProductId:   item.ProductID,
			ProductName: item.ProductName,
			Quantity:    item.Quantity,
			Price:       item.Price,
		})
	}

	return pbCart
}
