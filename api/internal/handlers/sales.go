package handlers

import (
	"encoding/json"
	"net/http"

	orderingpb "thugcorp.io/ordering/proto"
	"thugcorp.io/grocery/api/internal/middleware"
	"thugcorp.io/grocery/api/internal/respond"
)

type saleItemReq struct {
	ProductID  string `json:"product_id"`
	Qty        int64  `json:"qty"`
	PriceCents int64  `json:"price_cents"` // informational only; ordering service fetches live price
}

type confirmSaleReq struct {
	Items         []saleItemReq `json:"items"`
	DiscountCents int64         `json:"discount_cents"`
}

type confirmSaleResp struct {
	ID         string `json:"id"`
	TotalCents int64  `json:"total_cents"`
}

// POST /v1/sales — POS direct sale for admins
// Clears the admin's cart, adds items, checks out, and immediately confirms the order.
func (h *Handlers) ConfirmSale(w http.ResponseWriter, r *http.Request) {
	userID, _, _, _ := middleware.ClaimsFromCtx(r.Context())

	var body confirmSaleReq
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(body.Items) == 0 {
		respond.Error(w, http.StatusBadRequest, "cart is empty")
		return
	}

	ctx := h.outgoingCtx(r)

	// 1. Clear any leftover items in the admin's cart
	if _, err := h.svc.Ordering.ClearCart(ctx, &orderingpb.ClearCartRequest{UserId: userID}); err != nil {
		respond.GRPCError(w, err)
		return
	}

	// 2. Add each item; the ordering service fetches name, price, and business_id from the catalog
	for _, item := range body.Items {
		if _, err := h.svc.Ordering.AddItem(ctx, &orderingpb.AddItemRequest{
			UserId:    userID,
			ProductId: item.ProductID,
			Quantity:  item.Qty,
		}); err != nil {
			respond.GRPCError(w, err)
			return
		}
	}

	// 3. Checkout the cart → creates a pending order
	// A fresh idempotency key per sale avoids the UNIQUE constraint on a second sale.
	order, err := h.svc.Ordering.Checkout(ctx, &orderingpb.CheckoutRequest{
		UserId:         userID,
		PaymentMethod:  "cash",
		IdempotencyKey: randomHex(),
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}

	// 4. Immediately confirm — POS sales are paid at the counter
	order, err = h.svc.Ordering.UpdateOrderStatus(ctx, &orderingpb.UpdateOrderStatusRequest{
		OrderId: order.Id,
		Status:  orderingpb.OrderStatus_ORDER_STATUS_CONFIRMED,
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}

	respond.JSON(w, http.StatusCreated, confirmSaleResp{
		ID:         order.Id,
		TotalCents: order.Total,
	})
}
