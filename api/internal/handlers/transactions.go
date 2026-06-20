package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	transactionpb "thugcorp.io/grocery/transaction/proto"
	"thugcorp.io/grocery/api/internal/middleware"
	"thugcorp.io/grocery/api/internal/respond"
)

// GET /v1/transactions
func (h *Handlers) ListTransactions(w http.ResponseWriter, r *http.Request) {
	userID, _, businessID, _ := middleware.ClaimsFromCtx(r.Context())
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))

	resp, err := h.svc.Transactions.ListTransactions(h.outgoingCtx(r), &transactionpb.ListTransactionsRequest{
		UserId:     userID,
		BusinessId: businessID,
		Status:     r.URL.Query().Get("status"),
		Page:       int32(page),
		PageSize:   int32(pageSize),
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, resp)
}

// GET /v1/transactions/{id}
func (h *Handlers) GetTransaction(w http.ResponseWriter, r *http.Request) {
	resp, err := h.svc.Transactions.GetTransaction(h.outgoingCtx(r), &transactionpb.GetTransactionRequest{
		TransactionId: chi.URLParam(r, "id"),
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, resp)
}

// POST /v1/transactions
func (h *Handlers) CreateTransaction(w http.ResponseWriter, r *http.Request) {
	userID, _, businessID, _ := middleware.ClaimsFromCtx(r.Context())

	var body struct {
		Items []struct {
			ProductID   string  `json:"product_id"`
			ProductName string  `json:"product_name"`
			BusinessID  string  `json:"business_id"`
			Quantity    int32   `json:"quantity"`
			Price       float64 `json:"price"`
		} `json:"items"`
		PaymentMethod string `json:"payment_method"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	items := make([]*transactionpb.TransactionItem, 0, len(body.Items))
	for _, it := range body.Items {
		bid := it.BusinessID
		if bid == "" {
			bid = businessID
		}
		items = append(items, &transactionpb.TransactionItem{
			ProductId:   it.ProductID,
			ProductName: it.ProductName,
			BusinessId:  bid,
			Quantity:    it.Quantity,
			Price:       it.Price,
		})
	}

	resp, err := h.svc.Transactions.CreateTransaction(h.outgoingCtx(r), &transactionpb.CreateTransactionRequest{
		UserId:        userID,
		BusinessId:    businessID,
		Items:         items,
		PaymentMethod: body.PaymentMethod,
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusCreated, resp)
}

// PUT /v1/transactions/{id}/status
func (h *Handlers) UpdateTransactionStatus(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.svc.Transactions.UpdateTransactionStatus(h.outgoingCtx(r), &transactionpb.UpdateStatusRequest{
		TransactionId: chi.URLParam(r, "id"),
		Status:        body.Status,
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, resp)
}
