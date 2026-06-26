package handlers

import (
	"encoding/json"
	"net/http"

	catalogpb "thugcorp.io/catalog/proto"
	"thugcorp.io/grocery/api/internal/middleware"
	"thugcorp.io/grocery/api/internal/respond"
)

// GET /v1/inventory/low-stock-count — count of products with available <= threshold for caller's business
func (h *Handlers) LowStockCount(w http.ResponseWriter, r *http.Request) {
	_, _, businessID, _ := middleware.ClaimsFromCtx(r.Context())
	if businessID == "" {
		respond.JSON(w, http.StatusOK, map[string]int{"count": 0})
		return
	}

	const threshold = 10

	products, err := h.svc.Catalog.ListProducts(h.outgoingCtx(r), &catalogpb.ListProductsRequest{
		BusinessId: businessID,
		PageSize:   500,
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	if len(products.Products) == 0 {
		respond.JSON(w, http.StatusOK, map[string]int{"count": 0})
		return
	}

	queries := make([]*catalogpb.AvailabilityQuery, len(products.Products))
	for i, p := range products.Products {
		queries[i] = &catalogpb.AvailabilityQuery{
			ProductId: p.Id,
			Quantity:  threshold + 1, // !sufficient ⟺ available <= threshold
		}
	}

	avail, err := h.svc.Catalog.CheckAvailability(h.outgoingCtx(r), &catalogpb.CheckAvailabilityRequest{Items: queries})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}

	count := 0
	for _, result := range avail.Results {
		if !result.Sufficient {
			count++
		}
	}
	respond.JSON(w, http.StatusOK, map[string]int{"count": count})
}

// POST /v1/inventory/availability
func (h *Handlers) CheckAvailability(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Items []struct {
			ProductID  string `json:"product_id"`
			LocationID string `json:"location_id"`
			Quantity   int64  `json:"quantity"`
		} `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	queries := make([]*catalogpb.AvailabilityQuery, 0, len(body.Items))
	for _, item := range body.Items {
		queries = append(queries, &catalogpb.AvailabilityQuery{
			ProductId:  item.ProductID,
			LocationId: item.LocationID,
			Quantity:   item.Quantity,
		})
	}

	resp, err := h.svc.Catalog.CheckAvailability(h.outgoingCtx(r), &catalogpb.CheckAvailabilityRequest{
		Items: queries,
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, resp)
}

// GET /v1/inventory/{businessId} — use ListProducts scoped to business as stock view
func (h *Handlers) ListStock(w http.ResponseWriter, r *http.Request) {
	respond.Error(w, http.StatusNotImplemented, "use GET /v1/products?business_id=<id>")
}

// GET /v1/inventory/{businessId}/{productId}
func (h *Handlers) GetStock(w http.ResponseWriter, r *http.Request) {
	respond.Error(w, http.StatusNotImplemented, "use GET /v1/products/<id>")
}

// POST /v1/inventory/adjust
func (h *Handlers) AdjustStock(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ProductID       string `json:"product_id"`
		LocationID      string `json:"location_id"`
		Delta           *int64 `json:"delta"`
		SetTo           *int64 `json:"set_to"`
		Reason          string `json:"reason"`
		ExpectedVersion string `json:"expected_version"`
		IdempotencyKey  string `json:"idempotency_key"`
		UnitCost        int64  `json:"unit_cost"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req := &catalogpb.AdjustStockRequest{
		ProductId:       body.ProductID,
		LocationId:      body.LocationID,
		ExpectedVersion: body.ExpectedVersion,
		IdempotencyKey:  body.IdempotencyKey,
		UnitCost:        body.UnitCost,
	}
	switch {
	case body.Delta != nil:
		req.Change = &catalogpb.AdjustStockRequest_Delta{Delta: *body.Delta}
	case body.SetTo != nil:
		req.Change = &catalogpb.AdjustStockRequest_SetTo{SetTo: *body.SetTo}
	default:
		respond.Error(w, http.StatusBadRequest, "either delta or set_to is required")
		return
	}

	resp, err := h.svc.Catalog.AdjustStock(h.outgoingCtx(r), req)
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, resp)
}
