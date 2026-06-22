package internal

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"thugcorp.io/payment/internal/domain"
	"thugcorp.io/payment/internal/middleware"
	pb "thugcorp.io/payment/proto"
)

type PaymentHandler struct {
	pb.UnimplementedPaymentServiceServer
	svc PaymentService
}

func NewPaymentHandler(svc PaymentService) *PaymentHandler {
	return &PaymentHandler{svc: svc}
}

// ---- helpers ----

func businessIDFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(middleware.BusinessIDKey).(string)
	return v
}

func userIDFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(middleware.UserIDKey).(string)
	return v
}

func mapProvider(p domain.Provider) pb.Provider {
	switch p {
	case domain.ProviderMTNMomo:
		return pb.Provider_PROVIDER_MTN_MOMO
	case domain.ProviderAirtelMoney:
		return pb.Provider_PROVIDER_AIRTEL_MONEY
	default:
		return pb.Provider_PROVIDER_UNSPECIFIED
	}
}

func protoProvider(p pb.Provider) domain.Provider {
	switch p {
	case pb.Provider_PROVIDER_MTN_MOMO:
		return domain.ProviderMTNMomo
	case pb.Provider_PROVIDER_AIRTEL_MONEY:
		return domain.ProviderAirtelMoney
	default:
		return domain.ProviderUnspecified
	}
}

func mapPaymentStatus(s domain.PaymentStatus) pb.PaymentStatus {
	switch s {
	case domain.PaymentStatusPending:
		return pb.PaymentStatus_PAYMENT_STATUS_PENDING
	case domain.PaymentStatusSucceeded:
		return pb.PaymentStatus_PAYMENT_STATUS_SUCCEEDED
	case domain.PaymentStatusFailed:
		return pb.PaymentStatus_PAYMENT_STATUS_FAILED
	case domain.PaymentStatusRefunded:
		return pb.PaymentStatus_PAYMENT_STATUS_REFUNDED
	default:
		return pb.PaymentStatus_PAYMENT_STATUS_UNSPECIFIED
	}
}

func mapTxnType(t domain.TransactionType) pb.TransactionType {
	switch t {
	case domain.TransactionTypePayment:
		return pb.TransactionType_TRANSACTION_TYPE_PAYMENT
	case domain.TransactionTypeCommission:
		return pb.TransactionType_TRANSACTION_TYPE_COMMISSION
	case domain.TransactionTypePayout:
		return pb.TransactionType_TRANSACTION_TYPE_PAYOUT
	case domain.TransactionTypeRefund:
		return pb.TransactionType_TRANSACTION_TYPE_REFUND
	default:
		return pb.TransactionType_TRANSACTION_TYPE_UNSPECIFIED
	}
}

func mapPayment(p *domain.Payment) *pb.Payment {
	return &pb.Payment{
		Id:          p.ID,
		OrderId:     p.OrderID,
		UserId:      p.UserID,
		Amount:      p.Amount,
		Currency:    p.Currency,
		Provider:    mapProvider(p.Provider),
		ProviderRef: p.ProviderRef,
		Status:      mapPaymentStatus(p.Status),
		CreatedAt:   timestamppb.New(p.CreatedAt),
	}
}

func mapTransaction(t *domain.Transaction) *pb.Transaction {
	return &pb.Transaction{
		Id:         t.ID,
		PaymentId:  t.PaymentID,
		BusinessId: t.BusinessID,
		Type:       mapTxnType(t.Type),
		Amount:     t.Amount,
		Currency:   t.Currency,
		Note:       t.Note,
		CreatedAt:  timestamppb.New(t.CreatedAt),
	}
}

func mapPayout(p *domain.Payout) *pb.Payout {
	return &pb.Payout{
		Id:          p.ID,
		BusinessId:  p.BusinessID,
		Amount:      p.Amount,
		Currency:    p.Currency,
		Provider:    mapProvider(p.Provider),
		ProviderRef: p.ProviderRef,
		Status:      mapPaymentStatus(p.Status),
		CreatedAt:   timestamppb.New(p.CreatedAt),
	}
}

// ---- gRPC handlers ----

func (h *PaymentHandler) InitiatePayment(ctx context.Context, req *pb.InitiatePaymentRequest) (*pb.Payment, error) {
	if req.OrderId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "order_id is required")
	}
	userID := userIDFromCtx(ctx)
	if userID == "" {
		userID = req.UserId
	}

	p, err := h.svc.InitiatePayment(domain.InitiatePaymentInput{
		OrderID:        req.OrderId,
		UserID:         userID,
		Amount:         req.Amount,
		Currency:       req.Currency,
		Provider:       protoProvider(req.Provider),
		Phone:          req.Phone,
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return mapPayment(p), nil
}

func (h *PaymentHandler) GetPayment(ctx context.Context, req *pb.GetPaymentRequest) (*pb.Payment, error) {
	if req.PaymentId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "payment_id is required")
	}
	p, err := h.svc.GetPayment(req.PaymentId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "%v", err)
	}
	return mapPayment(p), nil
}

func (h *PaymentHandler) Refund(ctx context.Context, req *pb.RefundRequest) (*pb.Payment, error) {
	if req.PaymentId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "payment_id is required")
	}
	p, err := h.svc.Refund(domain.RefundInput{
		PaymentID:      req.PaymentId,
		Amount:         req.Amount,
		Reason:         req.Reason,
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return mapPayment(p), nil
}

func (h *PaymentHandler) ListTransactions(ctx context.Context, req *pb.ListTransactionsRequest) (*pb.ListTransactionsResponse, error) {
	txns, nextToken, err := h.svc.ListTransactions(domain.ListTransactionsFilter{
		BusinessID: req.BusinessId,
		PaymentID:  req.PaymentId,
		PageSize:   int(req.PageSize),
		PageToken:  req.PageToken,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}

	out := make([]*pb.Transaction, len(txns))
	for i := range txns {
		out[i] = mapTransaction(&txns[i])
	}
	return &pb.ListTransactionsResponse{
		Transactions:  out,
		NextPageToken: nextToken,
	}, nil
}

func (h *PaymentHandler) GetBalance(ctx context.Context, req *pb.GetBalanceRequest) (*pb.Balance, error) {
	if req.BusinessId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "business_id is required")
	}
	amount, currency, err := h.svc.GetBalance(req.BusinessId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return &pb.Balance{
		BusinessId: req.BusinessId,
		Amount:     amount,
		Currency:   currency,
	}, nil
}

func (h *PaymentHandler) InitiatePayout(ctx context.Context, req *pb.InitiatePayoutRequest) (*pb.Payout, error) {
	if req.BusinessId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "business_id is required")
	}
	p, err := h.svc.InitiatePayout(domain.InitiatePayoutInput{
		BusinessID:     req.BusinessId,
		Amount:         req.Amount,
		Currency:       req.Currency,
		Provider:       protoProvider(req.Provider),
		Phone:          req.Phone,
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return mapPayout(p), nil
}

// ---- HTTP webhook handler (mobile money callback) ----

type WebhookPayload struct {
	PaymentID   string `json:"payment_id"`
	Success     bool   `json:"success"`
	ProviderRef string `json:"provider_ref"`
	Timestamp   string `json:"timestamp"`
}

func WebhookHandler(svc PaymentService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var payload WebhookPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid payload", http.StatusBadRequest)
			return
		}
		if payload.PaymentID == "" {
			http.Error(w, "payment_id is required", http.StatusBadRequest)
			return
		}

		log.Printf("webhook received: payment_id=%s success=%v ref=%s ts=%s",
			payload.PaymentID, payload.Success, payload.ProviderRef, payload.Timestamp)

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		_ = ctx // repo calls are sync here; pass ctx if you add ctx support to repo

		if err := svc.HandleWebhook(payload.PaymentID, payload.Success, payload.ProviderRef); err != nil {
			log.Printf("webhook processing failed: %v", err)
			http.Error(w, "processing error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}
}
