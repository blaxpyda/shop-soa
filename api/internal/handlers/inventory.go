package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	inventorypb "thugcorp.io/grocery/inventory/proto"
	"thugcorp.io/grocery/api/internal/respond"
)

// POST /v1/inventory/availability
func (h *Handlers) CheckAvailability(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Items []struct {
			BusinessID string `json:"business_id"`
			ProductID  string `json:"product_id"`
			LocationID string `json:"location_id"`
			Quantity   int64  `json:"quantity"`
		} `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	queries := make([]*inventorypb.AvailabilityQuery, 0, len(body.Items))
	for _, item := range body.Items {
		queries = append(queries, &inventorypb.AvailabilityQuery{
			BusinessId: item.BusinessID,
			ProductId:  item.ProductID,
			LocationId: item.LocationID,
			Quantity:   item.Quantity,
		})
	}

	resp, err := h.svc.Inventory.CheckAvailability(h.outgoingCtx(r), &inventorypb.CheckAvailabilityRequest{
		Items: queries,
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, resp)
}

// GET /v1/inventory/{businessId}
func (h *Handlers) ListStock(w http.ResponseWriter, r *http.Request) {
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))

	resp, err := h.svc.Inventory.ListStock(h.outgoingCtx(r), &inventorypb.ListStockRequest{
		BusinessId:  chi.URLParam(r, "businessId"),
		LocationId:  r.URL.Query().Get("location_id"),
		PageSize:    int32(pageSize),
		PageToken:   r.URL.Query().Get("page_token"),
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, resp)
}

// GET /v1/inventory/{businessId}/{productId}
func (h *Handlers) GetStock(w http.ResponseWriter, r *http.Request) {
	resp, err := h.svc.Inventory.GetStock(h.outgoingCtx(r), &inventorypb.GetStockRequest{
		BusinessId: chi.URLParam(r, "businessId"),
		ProductId:  chi.URLParam(r, "productId"),
		LocationId: r.URL.Query().Get("location_id"),
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, resp)
}

// POST /v1/inventory/adjust
func (h *Handlers) AdjustStock(w http.ResponseWriter, r *http.Request) {
	var body struct {
		BusinessID      string `json:"business_id"`
		ProductID       string `json:"product_id"`
		LocationID      string `json:"location_id"`
		Delta           *int64 `json:"delta"`
		SetTo           *int64 `json:"set_to"`
		Reason          string `json:"reason"`
		Note            string `json:"note"`
		ExpectedVersion string `json:"expected_version"`
		IdempotencyKey  string `json:"idempotency_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req := &inventorypb.AdjustStockRequest{
		BusinessId:      body.BusinessID,
		ProductId:       body.ProductID,
		LocationId:      body.LocationID,
		Note:            body.Note,
		ExpectedVersion: body.ExpectedVersion,
		IdempotencyKey:  body.IdempotencyKey,
	}
	switch {
	case body.Delta != nil:
		req.Change = &inventorypb.AdjustStockRequest_Delta{Delta: *body.Delta}
	case body.SetTo != nil:
		req.Change = &inventorypb.AdjustStockRequest_SetTo{SetTo: *body.SetTo}
	default:
		respond.Error(w, http.StatusBadRequest, "either delta or set_to is required")
		return
	}

	resp, err := h.svc.Inventory.AdjustStock(h.outgoingCtx(r), req)
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, resp)
}
