package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	catalogpb "thugcorp.io/catalog/proto"
	"thugcorp.io/grocery/api/internal/middleware"
	"thugcorp.io/grocery/api/internal/respond"
)

type productItem struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Category   string `json:"category"`
	PriceCents int64  `json:"price_cents"`
	Currency   string `json:"currency"`
	Stock      int64  `json:"stock"`
	Active     bool   `json:"active"`
}

type productsListResp struct {
	Products []productItem `json:"products"`
}

func (h *Handlers) enrichWithStock(r *http.Request, prods []*catalogpb.Product) productsListResp {
	if len(prods) == 0 {
		return productsListResp{Products: []productItem{}}
	}
	queries := make([]*catalogpb.AvailabilityQuery, len(prods))
	for i, p := range prods {
		queries[i] = &catalogpb.AvailabilityQuery{ProductId: p.Id, Quantity: 1}
	}
	stockMap := map[string]int64{}
	if avail, err := h.svc.Catalog.CheckAvailability(h.outgoingCtx(r), &catalogpb.CheckAvailabilityRequest{Items: queries}); err == nil {
		for _, ar := range avail.Results {
			stockMap[ar.ProductId] = ar.Available
		}
	}
	items := make([]productItem, len(prods))
	for i, p := range prods {
		items[i] = productItem{
			ID:         p.Id,
			Name:       p.Name,
			Category:   p.Category,
			PriceCents: p.Price,
			Currency:   p.Currency,
			Stock:      stockMap[p.Id],
			Active:     p.Active,
		}
	}
	return productsListResp{Products: items}
}

func bizID(r *http.Request) string {
	if v := r.URL.Query().Get("business_id"); v != "" {
		return v
	}
	_, _, jwtBiz, _ := middleware.ClaimsFromCtx(r.Context())
	return jwtBiz
}

// GET /v1/products
func (h *Handlers) ListProducts(w http.ResponseWriter, r *http.Request) {
	resp, err := h.svc.Catalog.ListProducts(h.outgoingCtx(r), &catalogpb.ListProductsRequest{
		BusinessId: bizID(r),
		PageSize:   pageSize(r),
		PageToken:  r.URL.Query().Get("page_token"),
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, h.enrichWithStock(r, resp.Products))
}

// GET /v1/products/search?q=sugar
func (h *Handlers) SearchProducts(w http.ResponseWriter, r *http.Request) {
	resp, err := h.svc.Catalog.ListProducts(h.outgoingCtx(r), &catalogpb.ListProductsRequest{
		BusinessId: bizID(r),
		Query:      r.URL.Query().Get("q"),
		PageSize:   pageSize(r),
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, h.enrichWithStock(r, resp.Products))
}

// GET /v1/products/{id}
func (h *Handlers) GetProduct(w http.ResponseWriter, r *http.Request) {
	resp, err := h.svc.Catalog.GetProduct(h.outgoingCtx(r), &catalogpb.GetProductRequest{
		ProductId: chi.URLParam(r, "id"),
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, resp)
}

// POST /v1/products
func (h *Handlers) CreateProduct(w http.ResponseWriter, r *http.Request) {
	_, _, businessID, _ := middleware.ClaimsFromCtx(r.Context())

	var body struct {
		Name         string `json:"name"`
		Description  string `json:"description"`
		Category     string `json:"category"`
		Price        int64  `json:"price"`
		CostPrice    int64  `json:"cost_price"`
		Currency     string `json:"currency"`
		ImageURL     string `json:"image_url"`
		InitialStock int64  `json:"initial_stock"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Currency == "" {
		body.Currency = "UGX"
	}

	resp, err := h.svc.Catalog.CreateProduct(h.outgoingCtx(r), &catalogpb.CreateProductRequest{
		BusinessId:   businessID,
		Name:         body.Name,
		Description:  body.Description,
		Category:     body.Category,
		Price:        body.Price,
		CostPrice:    body.CostPrice,
		Currency:     body.Currency,
		ImageUrl:     body.ImageURL,
		InitialStock: body.InitialStock,
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusCreated, resp)
}

// PUT /v1/products/{id}
func (h *Handlers) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Category    string `json:"category"`
		Price       int64  `json:"price"`
		CostPrice   int64  `json:"cost_price"`
		Active      *bool  `json:"active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	active := true
	if body.Active != nil {
		active = *body.Active
	}

	resp, err := h.svc.Catalog.UpdateProduct(h.outgoingCtx(r), &catalogpb.UpdateProductRequest{
		ProductId:   chi.URLParam(r, "id"),
		Name:        body.Name,
		Description: body.Description,
		Category:    body.Category,
		Price:       body.Price,
		CostPrice:   body.CostPrice,
		Active:      active,
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, resp)
}

// DELETE /v1/products/{id}
func (h *Handlers) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	respond.Error(w, http.StatusNotImplemented, "delete product not supported")
}
