package handlers

import (
	"net/http"

	paymentpb "thugcorp.io/payment/proto"
	"thugcorp.io/grocery/api/internal/middleware"
	"thugcorp.io/grocery/api/internal/respond"
)

// GET /v1/transactions
func (h *Handlers) ListTransactions(w http.ResponseWriter, r *http.Request) {
	_, _, businessID, _ := middleware.ClaimsFromCtx(r.Context())

	resp, err := h.svc.Payment.ListTransactions(h.outgoingCtx(r), &paymentpb.ListTransactionsRequest{
		BusinessId: businessID,
		PaymentId:  r.URL.Query().Get("payment_id"),
		PageSize:   pageSize(r),
		PageToken:  r.URL.Query().Get("page_token"),
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, resp)
}

// GET /v1/transactions/{id} — no single-txn RPC in payment proto; use ListTransactions with payment_id
func (h *Handlers) GetTransaction(w http.ResponseWriter, r *http.Request) {
	respond.Error(w, http.StatusNotImplemented, "use GET /v1/transactions?payment_id=<id>")
}

// POST /v1/transactions — removed; use POST /v1/payments instead
func (h *Handlers) CreateTransaction(w http.ResponseWriter, r *http.Request) {
	respond.Error(w, http.StatusGone, "use POST /v1/payments to initiate a payment")
}

// PUT /v1/transactions/{id}/status — payments are updated via webhook
func (h *Handlers) UpdateTransactionStatus(w http.ResponseWriter, r *http.Request) {
	respond.Error(w, http.StatusGone, "payment status is updated via provider webhook")
}
