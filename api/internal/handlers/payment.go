package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	paymentpb "thugcorp.io/payment/proto"
	"thugcorp.io/grocery/api/internal/middleware"
	"thugcorp.io/grocery/api/internal/respond"
)

// POST /v1/payments
func (h *Handlers) InitiatePayment(w http.ResponseWriter, r *http.Request) {
	userID, _, _, _ := middleware.ClaimsFromCtx(r.Context())

	var body struct {
		OrderID        string `json:"order_id"`
		Amount         int64  `json:"amount"`
		Currency       string `json:"currency"`
		Provider       string `json:"provider"`
		Phone          string `json:"phone"`
		IdempotencyKey string `json:"idempotency_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.svc.Payment.InitiatePayment(h.outgoingCtx(r), &paymentpb.InitiatePaymentRequest{
		OrderId:        body.OrderID,
		UserId:         userID,
		Amount:         body.Amount,
		Currency:       body.Currency,
		Provider:       protoProvider(body.Provider),
		Phone:          body.Phone,
		IdempotencyKey: body.IdempotencyKey,
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusCreated, resp)
}

// GET /v1/payments/{id}
func (h *Handlers) GetPayment(w http.ResponseWriter, r *http.Request) {
	resp, err := h.svc.Payment.GetPayment(h.outgoingCtx(r), &paymentpb.GetPaymentRequest{
		PaymentId: chi.URLParam(r, "id"),
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, resp)
}

// POST /v1/payments/{id}/refund
func (h *Handlers) Refund(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Amount         int64  `json:"amount"`
		Reason         string `json:"reason"`
		IdempotencyKey string `json:"idempotency_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.svc.Payment.Refund(h.outgoingCtx(r), &paymentpb.RefundRequest{
		PaymentId:      chi.URLParam(r, "id"),
		Amount:         body.Amount,
		Reason:         body.Reason,
		IdempotencyKey: body.IdempotencyKey,
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, resp)
}

// GET /v1/payments/balance
func (h *Handlers) GetBalance(w http.ResponseWriter, r *http.Request) {
	_, _, businessID, _ := middleware.ClaimsFromCtx(r.Context())

	resp, err := h.svc.Payment.GetBalance(h.outgoingCtx(r), &paymentpb.GetBalanceRequest{
		BusinessId: businessID,
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, resp)
}

// POST /v1/payments/payouts
func (h *Handlers) InitiatePayout(w http.ResponseWriter, r *http.Request) {
	_, _, businessID, _ := middleware.ClaimsFromCtx(r.Context())

	var body struct {
		Amount         int64  `json:"amount"`
		Currency       string `json:"currency"`
		Provider       string `json:"provider"`
		Phone          string `json:"phone"`
		IdempotencyKey string `json:"idempotency_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.svc.Payment.InitiatePayout(h.outgoingCtx(r), &paymentpb.InitiatePayoutRequest{
		BusinessId:     businessID,
		Amount:         body.Amount,
		Currency:       body.Currency,
		Provider:       protoProvider(body.Provider),
		Phone:          body.Phone,
		IdempotencyKey: body.IdempotencyKey,
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusCreated, resp)
}

func protoProvider(s string) paymentpb.Provider {
	switch s {
	case "mtn_momo":
		return paymentpb.Provider_PROVIDER_MTN_MOMO
	case "airtel_money":
		return paymentpb.Provider_PROVIDER_AIRTEL_MONEY
	default:
		return paymentpb.Provider_PROVIDER_UNSPECIFIED
	}
}
