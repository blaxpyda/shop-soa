package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	productpb "thugcorp.io/grocery/product/proto"
	"thugcorp.io/grocery/api/internal/respond"
)

// GET /v1/products
func (h *Handlers) ListProducts(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))

	resp, err := h.svc.Products.ListProducts(h.outgoingCtx(r), &productpb.ListProductsRequest{
		Category: r.URL.Query().Get("category"),
		Page:     int32(page),
		PageSize: int32(pageSize),
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, resp)
}

// GET /v1/products/search
func (h *Handlers) SearchProducts(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))

	resp, err := h.svc.Products.SearchProducts(h.outgoingCtx(r), &productpb.SearchProductsRequest{
		Query:    r.URL.Query().Get("q"),
		Page:     int32(page),
		PageSize: int32(pageSize),
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, resp)
}

// GET /v1/products/{id}
func (h *Handlers) GetProduct(w http.ResponseWriter, r *http.Request) {
	resp, err := h.svc.Products.GetProduct(h.outgoingCtx(r), &productpb.GetProductRequest{
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
	var body struct {
		Name            string  `json:"name"`
		Category        string  `json:"category"`
		SalesPrice      float64 `json:"salesprice"`
		PurchasePrices  float64 `json:"purchaseprices"`
		Stock           int32   `json:"stock"`
		SupplierContact string  `json:"suppliercontact"`
		SupplierName    string  `json:"suppliername"`
		ImageURL        string  `json:"image_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.svc.Products.CreateProduct(h.outgoingCtx(r), &productpb.CreateProductRequest{
		Name:            body.Name,
		Category:        body.Category,
		Salesprice:      body.SalesPrice,
		Purchaseprices:  body.PurchasePrices,
		Stock:           body.Stock,
		Suppliercontact: body.SupplierContact,
		Suppliername:    body.SupplierName,
		ImageUrl:        body.ImageURL,
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
		Name     string  `json:"name"`
		Category string  `json:"category"`
		Price    float64 `json:"price"`
		Stock    int32   `json:"stock"`
		ImageURL string  `json:"image_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.svc.Products.UpdateProduct(h.outgoingCtx(r), &productpb.UpdateProductRequest{
		ProductId: chi.URLParam(r, "id"),
		Name:      body.Name,
		Category:  body.Category,
		Price:     body.Price,
		Stock:     body.Stock,
		ImageUrl:  body.ImageURL,
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, resp)
}

// DELETE /v1/products/{id}
func (h *Handlers) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	resp, err := h.svc.Products.DeleteProduct(h.outgoingCtx(r), &productpb.DeleteProductRequest{
		ProductId: chi.URLParam(r, "id"),
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, resp)
}
