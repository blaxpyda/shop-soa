package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	cartpb "thugcorp.io/grocery/cart/proto"
	"thugcorp.io/grocery/api/internal/middleware"
	"thugcorp.io/grocery/api/internal/respond"
)

// GET /v1/cart
func (h *Handlers) GetCart(w http.ResponseWriter, r *http.Request) {
	userID, _, businessID, _ := middleware.ClaimsFromCtx(r.Context())

	resp, err := h.svc.Cart.GetCart(h.outgoingCtx(r), &cartpb.GetCartRequest{
		UserId:     userID,
		BusinessId: businessID,
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, resp)
}

// POST /v1/cart/items
func (h *Handlers) AddToCart(w http.ResponseWriter, r *http.Request) {
	userID, _, businessID, _ := middleware.ClaimsFromCtx(r.Context())

	var body struct {
		ProductID string `json:"product_id"`
		Quantity  int32  `json:"quantity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.svc.Cart.AddToCart(h.outgoingCtx(r), &cartpb.AddToCartRequest{
		UserId:     userID,
		BusinessId: businessID,
		ProductId:  body.ProductID,
		Quantity:   body.Quantity,
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, resp)
}

// DELETE /v1/cart/items/{productId}
func (h *Handlers) RemoveFromCart(w http.ResponseWriter, r *http.Request) {
	userID, _, businessID, _ := middleware.ClaimsFromCtx(r.Context())

	resp, err := h.svc.Cart.RemoveFromCart(h.outgoingCtx(r), &cartpb.RemoveFromCartRequest{
		UserId:     userID,
		BusinessId: businessID,
		ProductId:  chi.URLParam(r, "productId"),
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, resp)
}

// DELETE /v1/cart
func (h *Handlers) ClearCart(w http.ResponseWriter, r *http.Request) {
	userID, _, businessID, _ := middleware.ClaimsFromCtx(r.Context())

	resp, err := h.svc.Cart.ClearCart(h.outgoingCtx(r), &cartpb.ClearCartRequest{
		UserId:     userID,
		BusinessId: businessID,
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, resp)
}
