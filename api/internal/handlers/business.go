package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"thugcorp.io/grocery/api/internal/middleware"
	"thugcorp.io/grocery/api/internal/respond"
	authpb "thugcorp.io/grocery/auth/proto"
	businesspb "thugcorp.io/grocery/business/proto"
)

// POST /v1/businesses
func (h *Handlers) CreateBusiness(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Email       string `json:"email"`
		Phone       string `json:"phone"`
		Address     string `json:"address"`
		City        string `json:"city"`
		Country     string `json:"country"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.svc.Business.CreateBusiness(h.outgoingCtx(r), &businesspb.CreateBusinessRequest{
		Name:        body.Name,
		Description: body.Description,
		Email:       body.Email,
		Phone:       body.Phone,
		Address:     body.Address,
		City:        body.City,
		Country:     body.Country,
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}

	// Stamp the new business ID onto the owner's user record so their next
	// login JWT includes the correct business_id claim.
	callerID, _, _, _ := middleware.ClaimsFromCtx(r.Context())
	if _, err := h.svc.Auth.UpdateUser(h.outgoingCtx(r), &authpb.UpdateUserRequest{
		UserId:     callerID,
		BusinessId: resp.Id,
	}); err != nil {
		// Non-fatal: business was created; log and continue.
	}

	respond.JSON(w, http.StatusCreated, resp)
}

// GET /v1/businesses/{id}
func (h *Handlers) GetBusiness(w http.ResponseWriter, r *http.Request) {
	resp, err := h.svc.Business.GetBusiness(h.outgoingCtx(r), &businesspb.GetBusinessRequest{
		BusinessId: chi.URLParam(r, "id"),
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, resp)
}

// PUT /v1/businesses/{id}
func (h *Handlers) UpdateBusiness(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Email       string `json:"email"`
		Phone       string `json:"phone"`
		Address     string `json:"address"`
		City        string `json:"city"`
		Country     string `json:"country"`
		IsActive    bool   `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.svc.Business.UpdateBusiness(h.outgoingCtx(r), &businesspb.UpdateBusinessRequest{
		BusinessId:  chi.URLParam(r, "id"),
		Name:        body.Name,
		Description: body.Description,
		Email:       body.Email,
		Phone:       body.Phone,
		Address:     body.Address,
		City:        body.City,
		Country:     body.Country,
		IsActive:    body.IsActive,
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, resp)
}

// PUT /v1/businesses/{id}/users
func (h *Handlers) AddUserToBusiness(w http.ResponseWriter, r *http.Request) {
	businessID := chi.URLParam(r, "id")
	callerID, callerRole, _, _ := middleware.ClaimsFromCtx(r.Context())

	biz, err := h.svc.Business.GetBusiness(h.outgoingCtx(r), &businesspb.GetBusinessRequest{BusinessId: businessID})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}

	if callerRole != "super-admin" && callerID != biz.OwnerId {
		respond.Error(w, http.StatusForbidden, "forbidden: you do not own this business")
		return
	}

	var body struct {
		Email string `json:"email"`
		Phone string `json:"phone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Email == "" && body.Phone == "" {
		respond.Error(w, http.StatusBadRequest, "email or phone is required")
		return
	}

	// Resolve the user ID by matching email or phone against registered accounts.
	users, err := h.svc.Auth.ListUsers(h.outgoingCtx(r), &authpb.ListUsersRequest{Page: 1, PageSize: 200})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	var userID string
	for _, u := range users.Users {
		if (body.Email != "" && u.Email == body.Email) || (body.Phone != "" && u.Phone == body.Phone) {
			userID = u.Id
			break
		}
	}
	if userID == "" {
		respond.Error(w, http.StatusNotFound, "no account found with that email or phone")
		return
	}

	resp, err := h.svc.Auth.UpdateUser(h.outgoingCtx(r), &authpb.UpdateUserRequest{
		UserId:     userID,
		BusinessId: businessID,
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, resp)
}

// DELETE /v1/businesses/{id}
func (h *Handlers) DeleteBusiness(w http.ResponseWriter, r *http.Request) {
	resp, err := h.svc.Business.DeleteBusiness(h.outgoingCtx(r), &businesspb.DeleteBusinessRequest{
		BusinessId: chi.URLParam(r, "id"),
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, resp)
}
