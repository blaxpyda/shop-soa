package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	orderingpb "thugcorp.io/ordering/proto"
	"thugcorp.io/grocery/api/internal/middleware"
	"thugcorp.io/grocery/api/internal/respond"
)

// GET /v1/cart
func (h *Handlers) GetCart(w http.ResponseWriter, r *http.Request) {
	userID, _, _, _ := middleware.ClaimsFromCtx(r.Context())

	resp, err := h.svc.Ordering.GetCart(h.outgoingCtx(r), &orderingpb.GetCartRequest{
		UserId: userID,
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, resp)
}

// POST /v1/cart/items
func (h *Handlers) AddToCart(w http.ResponseWriter, r *http.Request) {
	userID, _, _, _ := middleware.ClaimsFromCtx(r.Context())

	var body struct {
		ProductID string `json:"product_id"`
		Quantity  int64  `json:"quantity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.svc.Ordering.AddItem(h.outgoingCtx(r), &orderingpb.AddItemRequest{
		UserId:    userID,
		ProductId: body.ProductID,
		Quantity:  body.Quantity,
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, resp)
}

// PUT /v1/cart/items/{productId}
func (h *Handlers) UpdateCartItem(w http.ResponseWriter, r *http.Request) {
	userID, _, _, _ := middleware.ClaimsFromCtx(r.Context())

	var body struct {
		Quantity int64 `json:"quantity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.svc.Ordering.UpdateItem(h.outgoingCtx(r), &orderingpb.UpdateItemRequest{
		UserId:    userID,
		ProductId: chi.URLParam(r, "productId"),
		Quantity:  body.Quantity,
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, resp)
}

// DELETE /v1/cart/items/{productId}
func (h *Handlers) RemoveFromCart(w http.ResponseWriter, r *http.Request) {
	userID, _, _, _ := middleware.ClaimsFromCtx(r.Context())

	resp, err := h.svc.Ordering.RemoveItem(h.outgoingCtx(r), &orderingpb.RemoveItemRequest{
		UserId:    userID,
		ProductId: chi.URLParam(r, "productId"),
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, resp)
}

// DELETE /v1/cart
func (h *Handlers) ClearCart(w http.ResponseWriter, r *http.Request) {
	userID, _, _, _ := middleware.ClaimsFromCtx(r.Context())

	resp, err := h.svc.Ordering.ClearCart(h.outgoingCtx(r), &orderingpb.ClearCartRequest{
		UserId: userID,
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, resp)
}
