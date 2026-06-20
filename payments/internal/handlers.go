package internal

import (
	"context"
	"errors"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"thugcorp.io/grocery/payment/internal/domain"
	"thugcorp.io/grocery/payment/internal/middleware"
	pb "thugcorp.io/grocery/payment/proto"
)

type paymentHandler struct {
	pb.UnimplementedPaymentServiceServer
	svc PaymentService
}

func NewPaymentHandler(svc PaymentService) *paymentHandler {
	return &paymentHandler{svc: svc}
}

func (h *paymentHandler) InitiatePayment(ctx context.Context, req *pb.InitiatePaymentRequest) (*pb.PaymentResponse, error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	p, err := h.svc.InitiatePayment(ctx, domain.CreatePaymentInput{
		OrderID:       req.OrderId,
		UserID:        userID,
		BusinessID:    req.BusinessId,
		Amount:        req.Amount,
		Currency:      req.Currency,
		PaymentMethod: req.PaymentMethod,
		Provider:      req.Provider,
		Metadata:      req.Metadata,
	})
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			return nil, status.Errorf(codes.InvalidArgument, "%v", err)
		}
		return nil, status.Errorf(codes.Internal, "failed to initiate payment: %v", err)
	}
	return toProto(p), nil
}

func (h *paymentHandler) ConfirmPayment(ctx context.Context, req *pb.ConfirmPaymentRequest) (*pb.PaymentResponse, error) {
	p, err := h.svc.ConfirmPayment(ctx, req.PaymentId, req.TransactionRef, req.Provider)
	if err != nil {
		return nil, grpcErr("confirm payment", err)
	}
	return toProto(p), nil
}

func (h *paymentHandler) GetPaymentStatus(ctx context.Context, req *pb.GetPaymentRequest) (*pb.PaymentResponse, error) {
	p, err := h.svc.GetPaymentStatus(ctx, req.PaymentId)
	if err != nil {
		return nil, grpcErr("get payment", err)
	}
	return toProto(p), nil
}

func (h *paymentHandler) RefundPayment(ctx context.Context, req *pb.RefundRequest) (*pb.RefundResponse, error) {
	p, err := h.svc.RefundPayment(ctx, req.PaymentId, req.Amount, req.Reason)
	if err != nil {
		return nil, grpcErr("refund payment", err)
	}
	return &pb.RefundResponse{
		Success:  true,
		RefundId: p.ID,
		Status:   p.Status,
	}, nil
}

func (h *paymentHandler) ListPayments(ctx context.Context, req *pb.ListPaymentsRequest) (*pb.ListPaymentsResponse, error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// use request user_id only if provided (admin querying another user), otherwise scope to caller
	targetUserID := req.UserId
	if targetUserID == "" {
		targetUserID = userID
	}

	payments, total, err := h.svc.ListPayments(ctx, domain.ListFilter{
		UserID:     targetUserID,
		BusinessID: req.BusinessId,
		Status:     req.Status,
		Page:       req.Page,
		PageSize:   req.PageSize,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list payments: %v", err)
	}

	resp := &pb.ListPaymentsResponse{
		Total:   int32(total),
		HasMore: int64(req.Page)*int64(req.PageSize) < total,
	}
	for _, p := range payments {
		resp.Payments = append(resp.Payments, toProto(p))
	}
	return resp, nil
}

// ---- helpers ----

func getUserIDFromContext(ctx context.Context) (string, error) {
	userID, ok := ctx.Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		return "", status.Errorf(codes.Unauthenticated, "user ID not found in context")
	}
	return userID, nil
}

func grpcErr(op string, err error) error {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return status.Errorf(codes.NotFound, "%v", err)
	case errors.Is(err, domain.ErrInvalidInput):
		return status.Errorf(codes.InvalidArgument, "%v", err)
	case errors.Is(err, domain.ErrInvalidState):
		return status.Errorf(codes.FailedPrecondition, "%v", err)
	default:
		return status.Errorf(codes.Internal, "failed to %s: %v", op, err)
	}
}

func toProto(p *domain.Payment) *pb.PaymentResponse {
	return &pb.PaymentResponse{
		Id:             p.ID,
		OrderId:        p.OrderID,
		UserId:         p.UserID,
		BusinessId:     p.BusinessID,
		Amount:         p.Amount,
		Currency:       p.Currency,
		Status:         p.Status,
		PaymentMethod:  p.PaymentMethod,
		Provider:       p.Provider,
		TransactionRef: p.TransactionRef,
		ErrorMessage:   p.ErrorMessage,
		CreatedAt:      p.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      p.UpdatedAt.Format(time.RFC3339),
	}
}
