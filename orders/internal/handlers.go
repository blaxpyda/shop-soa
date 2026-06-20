package internal

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"thugcorp.io/grocery/order/internal/domain"
	"thugcorp.io/grocery/order/internal/middleware"
	pb "thugcorp.io/grocery/order/proto"
)

type orderHandler struct {
	pb.UnimplementedOrderServiceServer
	svc OrderService
}

func NewOrderHandler(svc OrderService) *orderHandler {
	return &orderHandler{svc: svc}
}

func (h *orderHandler) CreateOrder(ctx context.Context, req *pb.CreateOrderRequest) (*pb.Order, error) {
	callerID, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	o, err := h.svc.CreateOrder(ctx, domain.CreateOrderInput{
		UserID:          callerID,
		BusinessID:      req.BusinessId,
		ShippingAddress: req.ShippingAddress,
		PaymentMethod:   req.PaymentMethod,
	})
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}
	return mapToProto(o), nil
}

func (h *orderHandler) GetOrder(ctx context.Context, req *pb.GetOrderRequest) (*pb.Order, error) {
	callerID, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	callerRole, _ := ctx.Value(middleware.RoleKey).(string)

	o, err := h.svc.GetOrder(ctx, callerID, callerRole, req.OrderId)
	if err != nil {
		switch err.Error() {
		case "order not found":
			return nil, status.Errorf(codes.NotFound, "%v", err)
		case "forbidden: you do not have access to this order":
			return nil, status.Errorf(codes.PermissionDenied, "%v", err)
		default:
			return nil, status.Errorf(codes.Internal, "%v", err)
		}
	}
	return mapToProto(o), nil
}

func (h *orderHandler) UpdateOrderStatus(ctx context.Context, req *pb.UpdateOrderStatusRequest) (*pb.Order, error) {
	_, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	callerRole, _ := ctx.Value(middleware.RoleKey).(string)

	o, err := h.svc.UpdateOrderStatus(ctx, callerRole, req.OrderId, req.Status)
	if err != nil {
		switch err.Error() {
		case "order not found":
			return nil, status.Errorf(codes.NotFound, "%v", err)
		case "forbidden: only admins can update order status":
			return nil, status.Errorf(codes.PermissionDenied, "%v", err)
		case "invalid order status":
			return nil, status.Errorf(codes.InvalidArgument, "%v", err)
		default:
			return nil, status.Errorf(codes.Internal, "%v", err)
		}
	}
	return mapToProto(o), nil
}

func (h *orderHandler) ListOrders(ctx context.Context, req *pb.ListOrdersRequest) (*pb.ListOrdersResponse, error) {
	callerID, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	callerRole, _ := ctx.Value(middleware.RoleKey).(string)

	// Non-admins can only list their own orders
	userID := req.UserId
	if callerRole != "admin" && callerRole != "super-admin" {
		userID = callerID
	}

	orders, total, err := h.svc.ListOrders(ctx, userID, req.BusinessId, req.Status, req.Page, req.PageSize)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}

	pbOrders := make([]*pb.Order, 0, len(orders))
	for _, o := range orders {
		pbOrders = append(pbOrders, mapToProto(o))
	}
	return &pb.ListOrdersResponse{Orders: pbOrders, Total: total}, nil
}

func (h *orderHandler) CancelOrder(ctx context.Context, req *pb.CancelOrderRequest) (*pb.Order, error) {
	callerID, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	callerRole, _ := ctx.Value(middleware.RoleKey).(string)

	o, err := h.svc.CancelOrder(ctx, callerID, callerRole, req.OrderId, req.Reason)
	if err != nil {
		switch err.Error() {
		case "order not found":
			return nil, status.Errorf(codes.NotFound, "%v", err)
		case "forbidden: you cannot cancel this order":
			return nil, status.Errorf(codes.PermissionDenied, "%v", err)
		case "order is already cancelled", "delivered orders cannot be cancelled":
			return nil, status.Errorf(codes.FailedPrecondition, "%v", err)
		default:
			return nil, status.Errorf(codes.Internal, "%v", err)
		}
	}
	return mapToProto(o), nil
}

// ---- Helpers ----

func userIDFromCtx(ctx context.Context) (string, error) {
	userID, ok := ctx.Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		return "", status.Errorf(codes.Unauthenticated, "user ID not found in context")
	}
	return userID, nil
}

func mapToProto(o *domain.Order) *pb.Order {
	items := make([]*pb.OrderItem, 0, len(o.Items))
	for _, item := range o.Items {
		items = append(items, &pb.OrderItem{
			ProductId:   item.ProductID,
			ProductName: item.ProductName,
			Quantity:    item.Quantity,
			Price:       item.Price,
		})
	}
	return &pb.Order{
		Id:              o.ID,
		UserId:          o.UserID,
		BusinessId:      o.BusinessID,
		Items:           items,
		TotalAmount:     o.TotalAmount,
		Status:          o.Status,
		ShippingAddress: o.ShippingAddress,
		PaymentMethod:   o.PaymentMethod,
	}
}
