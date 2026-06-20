package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	orderpb "thugcorp.io/grocery/order/proto"
	"thugcorp.io/grocery/api/internal/middleware"
	"thugcorp.io/grocery/api/internal/respond"
)

// POST /v1/orders
func (h *Handlers) CreateOrder(w http.ResponseWriter, r *http.Request) {
	userID, _, businessID, _ := middleware.ClaimsFromCtx(r.Context())

	var body struct {
		ShippingAddress string `json:"shipping_address"`
		PaymentMethod   string `json:"payment_method"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.svc.Orders.CreateOrder(h.outgoingCtx(r), &orderpb.CreateOrderRequest{
		UserId:          userID,
		BusinessId:      businessID,
		ShippingAddress: body.ShippingAddress,
		PaymentMethod:   body.PaymentMethod,
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusCreated, resp)
}

// GET /v1/orders
func (h *Handlers) ListOrders(w http.ResponseWriter, r *http.Request) {
	userID, _, businessID, _ := middleware.ClaimsFromCtx(r.Context())
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))

	resp, err := h.svc.Orders.ListOrders(h.outgoingCtx(r), &orderpb.ListOrdersRequest{
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

// GET /v1/orders/{id}
func (h *Handlers) GetOrder(w http.ResponseWriter, r *http.Request) {
	resp, err := h.svc.Orders.GetOrder(h.outgoingCtx(r), &orderpb.GetOrderRequest{
		OrderId: chi.URLParam(r, "id"),
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, resp)
}

// PUT /v1/orders/{id}/status
func (h *Handlers) UpdateOrderStatus(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.svc.Orders.UpdateOrderStatus(h.outgoingCtx(r), &orderpb.UpdateOrderStatusRequest{
		OrderId: chi.URLParam(r, "id"),
		Status:  body.Status,
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, resp)
}

// POST /v1/orders/{id}/cancel
func (h *Handlers) CancelOrder(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Reason string `json:"reason"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	resp, err := h.svc.Orders.CancelOrder(h.outgoingCtx(r), &orderpb.CancelOrderRequest{
		OrderId: chi.URLParam(r, "id"),
		Reason:  body.Reason,
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, resp)
}
