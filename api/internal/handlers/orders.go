package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	orderingpb "thugcorp.io/ordering/proto"
	"thugcorp.io/grocery/api/internal/middleware"
	"thugcorp.io/grocery/api/internal/respond"
)

// POST /v1/orders  (checkout: turns the cart into an order)
func (h *Handlers) CreateOrder(w http.ResponseWriter, r *http.Request) {
	userID, _, _, _ := middleware.ClaimsFromCtx(r.Context())

	var body struct {
		DeliveryAddress string `json:"delivery_address"`
		PaymentMethod   string `json:"payment_method"`
		IdempotencyKey  string `json:"idempotency_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.svc.Ordering.Checkout(h.outgoingCtx(r), &orderingpb.CheckoutRequest{
		UserId:          userID,
		DeliveryAddress: body.DeliveryAddress,
		PaymentMethod:   body.PaymentMethod,
		IdempotencyKey:  body.IdempotencyKey,
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

	req := &orderingpb.ListOrdersRequest{
		PageSize:  pageSize(r),
		PageToken: r.URL.Query().Get("page_token"),
	}
	// Prefer business filter when the caller belongs to a business
	if businessID != "" {
		req.Filter = &orderingpb.ListOrdersRequest_BusinessId{BusinessId: businessID}
	} else {
		req.Filter = &orderingpb.ListOrdersRequest_UserId{UserId: userID}
	}

	resp, err := h.svc.Ordering.ListOrders(h.outgoingCtx(r), req)
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, resp)
}

// GET /v1/orders/{id}
func (h *Handlers) GetOrder(w http.ResponseWriter, r *http.Request) {
	resp, err := h.svc.Ordering.GetOrder(h.outgoingCtx(r), &orderingpb.GetOrderRequest{
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

	resp, err := h.svc.Ordering.UpdateOrderStatus(h.outgoingCtx(r), &orderingpb.UpdateOrderStatusRequest{
		OrderId: chi.URLParam(r, "id"),
		Status:  protoOrderStatus(body.Status),
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, resp)
}

// POST /v1/orders/{id}/cancel
func (h *Handlers) CancelOrder(w http.ResponseWriter, r *http.Request) {
	resp, err := h.svc.Ordering.UpdateOrderStatus(h.outgoingCtx(r), &orderingpb.UpdateOrderStatusRequest{
		OrderId: chi.URLParam(r, "id"),
		Status:  orderingpb.OrderStatus_ORDER_STATUS_CANCELLED,
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, resp)
}

func protoOrderStatus(s string) orderingpb.OrderStatus {
	switch s {
	case "pending_payment":
		return orderingpb.OrderStatus_ORDER_STATUS_PENDING_PAYMENT
	case "confirmed":
		return orderingpb.OrderStatus_ORDER_STATUS_CONFIRMED
	case "preparing":
		return orderingpb.OrderStatus_ORDER_STATUS_PREPARING
	case "out_for_delivery":
		return orderingpb.OrderStatus_ORDER_STATUS_OUT_FOR_DELIVERY
	case "delivered":
		return orderingpb.OrderStatus_ORDER_STATUS_DELIVERED
	case "cancelled":
		return orderingpb.OrderStatus_ORDER_STATUS_CANCELLED
	default:
		return orderingpb.OrderStatus_ORDER_STATUS_UNSPECIFIED
	}
}
