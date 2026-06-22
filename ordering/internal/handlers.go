package internal

import (
	"context"
	"strconv"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"thugcorp.io/ordering/internal/domain"
	"thugcorp.io/ordering/internal/middleware"
	pb "thugcorp.io/ordering/proto"
)

type orderingHandler struct {
	pb.UnimplementedOrderingServiceServer
	svc OrderingService
}

func NewOrderingHandler(svc OrderingService) *orderingHandler {
	return &orderingHandler{svc: svc}
}

// ---- Cart ----

func (h *orderingHandler) GetCart(ctx context.Context, req *pb.GetCartRequest) (*pb.Cart, error) {
	userID := userIDFromCtx(ctx)
	if userID == "" {
		userID = req.UserId
	}

	cart, err := h.svc.GetCart(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return mapCart(cart), nil
}

func (h *orderingHandler) AddItem(ctx context.Context, req *pb.AddItemRequest) (*pb.Cart, error) {
	userID := userIDFromCtx(ctx)
	if userID == "" {
		userID = req.UserId
	}

	cart, err := h.svc.AddItem(ctx, userID, req.ProductId, req.Quantity)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}
	return mapCart(cart), nil
}

func (h *orderingHandler) UpdateItem(ctx context.Context, req *pb.UpdateItemRequest) (*pb.Cart, error) {
	userID := userIDFromCtx(ctx)
	if userID == "" {
		userID = req.UserId
	}

	cart, err := h.svc.UpdateItem(ctx, userID, req.ProductId, req.Quantity)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}
	return mapCart(cart), nil
}

func (h *orderingHandler) RemoveItem(ctx context.Context, req *pb.RemoveItemRequest) (*pb.Cart, error) {
	userID := userIDFromCtx(ctx)
	if userID == "" {
		userID = req.UserId
	}

	cart, err := h.svc.RemoveItem(ctx, userID, req.ProductId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return mapCart(cart), nil
}

func (h *orderingHandler) ClearCart(ctx context.Context, req *pb.ClearCartRequest) (*pb.ClearCartResponse, error) {
	userID := userIDFromCtx(ctx)
	if userID == "" {
		userID = req.UserId
	}

	if err := h.svc.ClearCart(ctx, userID); err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return &pb.ClearCartResponse{Cleared: true}, nil
}

// ---- Orders ----

func (h *orderingHandler) Checkout(ctx context.Context, req *pb.CheckoutRequest) (*pb.Order, error) {
	userID := userIDFromCtx(ctx)
	if userID == "" {
		userID = req.UserId
	}

	input := domain.CheckoutInput{
		UserID:          userID,
		DeliveryAddress: req.DeliveryAddress,
		PaymentMethod:   req.PaymentMethod,
		IdempotencyKey:  req.IdempotencyKey,
	}

	order, err := h.svc.Checkout(ctx, input)
	if err != nil {
		if err.Error() == "cart is empty" {
			return nil, status.Errorf(codes.FailedPrecondition, "%v", err)
		}
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return mapOrder(order), nil
}

func (h *orderingHandler) GetOrder(ctx context.Context, req *pb.GetOrderRequest) (*pb.Order, error) {
	order, err := h.svc.GetOrder(ctx, req.OrderId)
	if err != nil {
		if err.Error() == "order not found" {
			return nil, status.Errorf(codes.NotFound, "%v", err)
		}
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return mapOrder(order), nil
}

func (h *orderingHandler) ListOrders(ctx context.Context, req *pb.ListOrdersRequest) (*pb.ListOrdersResponse, error) {
	filter := domain.ListOrdersFilter{
		Status:    protoStatusToDomain(req.Status),
		PageSize:  int(req.PageSize),
		PageToken: req.PageToken,
	}

	switch f := req.Filter.(type) {
	case *pb.ListOrdersRequest_UserId:
		filter.UserID = f.UserId
	case *pb.ListOrdersRequest_BusinessId:
		filter.BusinessID = f.BusinessId
	default:
		// Fall back to the calling user's own orders.
		filter.UserID = userIDFromCtx(ctx)
	}

	orders, err := h.svc.ListOrders(ctx, filter)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}

	resp := &pb.ListOrdersResponse{}
	for _, o := range orders {
		resp.Orders = append(resp.Orders, mapOrder(o))
	}

	pageSize := int(req.PageSize)
	if pageSize > 0 && len(orders) == pageSize {
		offset := 0
		if req.PageToken != "" {
			if v, err := strconv.Atoi(req.PageToken); err == nil {
				offset = v
			}
		}
		resp.NextPageToken = strconv.Itoa(offset + pageSize)
	}

	return resp, nil
}

func (h *orderingHandler) UpdateOrderStatus(ctx context.Context, req *pb.UpdateOrderStatusRequest) (*pb.Order, error) {
	domainStatus := protoStatusToDomain(req.Status)
	order, err := h.svc.UpdateOrderStatus(ctx, req.OrderId, domainStatus)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return mapOrder(order), nil
}

// ---- Helpers ----

func userIDFromCtx(ctx context.Context) string {
	uid, _ := ctx.Value(middleware.UserIDKey).(string)
	return uid
}

func mapCart(c *domain.Cart) *pb.Cart {
	resp := &pb.Cart{
		UserId:   c.UserID,
		Subtotal: c.Subtotal(),
		Currency: c.DisplayCurrency(),
	}
	for _, item := range c.Items {
		resp.Items = append(resp.Items, &pb.CartItem{
			ProductId:  item.ProductID,
			BusinessId: item.BusinessID,
			Name:       item.Name,
			UnitPrice:  item.UnitPrice,
			Quantity:   item.Quantity,
		})
	}
	return resp
}

func mapOrder(o *domain.Order) *pb.Order {
	resp := &pb.Order{
		Id:        o.ID,
		UserId:    o.UserID,
		Status:    domainStatusToProto(o.Status),
		Total:     o.Total,
		Currency:  o.Currency,
		PaymentId: o.PaymentID,
		CreatedAt: timestamppb.New(o.CreatedAt),
	}
	for _, item := range o.Items {
		resp.Items = append(resp.Items, &pb.OrderItem{
			ProductId:   item.ProductID,
			BusinessId:  item.BusinessID,
			ProductName: item.ProductName,
			UnitPrice:   item.UnitPrice,
			Quantity:    item.Quantity,
			LineTotal:   item.LineTotal,
		})
	}
	return resp
}

func domainStatusToProto(s domain.OrderStatus) pb.OrderStatus {
	switch s {
	case domain.OrderStatusPendingPayment:
		return pb.OrderStatus_ORDER_STATUS_PENDING_PAYMENT
	case domain.OrderStatusConfirmed:
		return pb.OrderStatus_ORDER_STATUS_CONFIRMED
	case domain.OrderStatusPreparing:
		return pb.OrderStatus_ORDER_STATUS_PREPARING
	case domain.OrderStatusOutForDelivery:
		return pb.OrderStatus_ORDER_STATUS_OUT_FOR_DELIVERY
	case domain.OrderStatusDelivered:
		return pb.OrderStatus_ORDER_STATUS_DELIVERED
	case domain.OrderStatusCancelled:
		return pb.OrderStatus_ORDER_STATUS_CANCELLED
	default:
		return pb.OrderStatus_ORDER_STATUS_UNSPECIFIED
	}
}

func protoStatusToDomain(s pb.OrderStatus) domain.OrderStatus {
	switch s {
	case pb.OrderStatus_ORDER_STATUS_PENDING_PAYMENT:
		return domain.OrderStatusPendingPayment
	case pb.OrderStatus_ORDER_STATUS_CONFIRMED:
		return domain.OrderStatusConfirmed
	case pb.OrderStatus_ORDER_STATUS_PREPARING:
		return domain.OrderStatusPreparing
	case pb.OrderStatus_ORDER_STATUS_OUT_FOR_DELIVERY:
		return domain.OrderStatusOutForDelivery
	case pb.OrderStatus_ORDER_STATUS_DELIVERED:
		return domain.OrderStatusDelivered
	case pb.OrderStatus_ORDER_STATUS_CANCELLED:
		return domain.OrderStatusCancelled
	default:
		return domain.OrderStatusUnspecified
	}
}
