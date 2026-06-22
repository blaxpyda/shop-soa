package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	catalogpb "thugcorp.io/catalog/proto"
	"thugcorp.io/grocery/api/internal/middleware"
	"thugcorp.io/grocery/api/internal/respond"
)

// GET /v1/products
func (h *Handlers) ListProducts(w http.ResponseWriter, r *http.Request) {
	resp, err := h.svc.Catalog.ListProducts(h.outgoingCtx(r), &catalogpb.ListProductsRequest{
		BusinessId: r.URL.Query().Get("business_id"),
		PageSize:   pageSize(r),
		PageToken:  r.URL.Query().Get("page_token"),
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, resp)
}

// GET /v1/products/search — not in catalog proto; forward as ListProducts with business filter
func (h *Handlers) SearchProducts(w http.ResponseWriter, r *http.Request) {
	resp, err := h.svc.Catalog.ListProducts(h.outgoingCtx(r), &catalogpb.ListProductsRequest{
		BusinessId: r.URL.Query().Get("business_id"),
		PageSize:   pageSize(r),
		PageToken:  r.URL.Query().Get("page_token"),
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, resp)
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
		Currency     string `json:"currency"`
		ImageURL     string `json:"image_url"`
		InitialStock int64  `json:"initial_stock"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.svc.Catalog.CreateProduct(h.outgoingCtx(r), &catalogpb.CreateProductRequest{
		BusinessId:   businessID,
		Name:         body.Name,
		Description:  body.Description,
		Category:     body.Category,
		Price:        body.Price,
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
		Active:      active,
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, resp)
}

// DELETE /v1/products/{id} — catalog has no DeleteProduct RPC; return 204
func (h *Handlers) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	respond.Error(w, http.StatusNotImplemented, "delete product not supported")
}
