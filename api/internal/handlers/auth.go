package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"thugcorp.io/grocery/api/internal/respond"
	authpb "thugcorp.io/grocery/auth/proto"
)

// POST /v1/auth/signup
func (h *Handlers) Signup(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Phone    string `json:"phone"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.svc.Auth.Signup(r.Context(), &authpb.SignupRequest{
		Email:    body.Email,
		Phone:    body.Phone,
		Password: body.Password,
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusCreated, resp)
}

// POST /v1/auth/login
func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Phone    string `json:"phone"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.svc.Auth.Login(r.Context(), &authpb.LoginRequest{
		Email:    body.Email,
		Phone:    body.Phone,
		Password: body.Password,
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, resp)
}

// POST /v1/auth/verify
func (h *Handlers) VerifyCode(w http.ResponseWriter, r *http.Request) {
	var body struct {
		UserID string `json:"user_id"`
		Code   string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.svc.Auth.VerifyCode(r.Context(), &authpb.VerifyCodeRequest{
		UserId: body.UserID,
		Code:   body.Code,
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, resp)
}

// POST /v1/auth/resend
func (h *Handlers) ResendCode(w http.ResponseWriter, r *http.Request) {
	var body struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.svc.Auth.ResendCode(r.Context(), &authpb.ResendCodeRequest{
		UserId: body.UserID,
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, resp)
}

// GET /v1/auth/profile
func (h *Handlers) GetProfile(w http.ResponseWriter, r *http.Request) {
	resp, err := h.svc.Auth.GetProfile(h.outgoingCtx(r), &authpb.GetProfileRequest{})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, resp)
}

// PUT /v1/auth/profile
func (h *Handlers) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	var body struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Email     string `json:"email"`
		Phone     string `json:"phone"`
		Address   string `json:"address"`
		City      string `json:"city"`
		Country   string `json:"country"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.svc.Auth.UpdateProfile(h.outgoingCtx(r), &authpb.UpdateProfileRequest{
		FirstName: body.FirstName,
		LastName:  body.LastName,
		Email:     body.Email,
		Phone:     body.Phone,
		Address:   body.Address,
		City:      body.City,
		Country:   body.Country,
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, resp)
}

// POST /v1/admin/users — admin creates a staff user and assigns them to a business.
func (h *Handlers) AdminCreateUser(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name       string `json:"name"`
		Email      string `json:"email"`
		BusinessID string `json:"business_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Email == "" {
		respond.Error(w, http.StatusBadRequest, "email is required")
		return
	}
	if body.BusinessID == "" {
		respond.Error(w, http.StatusBadRequest, "business_id is required")
		return
	}

	parts := strings.SplitN(strings.TrimSpace(body.Name), " ", 2)
	firstName, lastName := "", ""
	if len(parts) > 0 {
		firstName = parts[0]
	}
	if len(parts) > 1 {
		lastName = parts[1]
	}

	resp, err := h.svc.Auth.CreateUser(h.outgoingCtx(r), &authpb.CreateUserRequest{
		Email:      body.Email,
		FirstName:  firstName,
		LastName:   lastName,
		BusinessId: body.BusinessID,
		Role:       "user",
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusCreated, resp)
}

// POST /v1/users
func (h *Handlers) CreateUser(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email     string `json:"email"`
		Phone     string `json:"phone"`
		Password  string `json:"password"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Role      string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.svc.Auth.CreateUser(h.outgoingCtx(r), &authpb.CreateUserRequest{
		Email:     body.Email,
		Phone:     body.Phone,
		Password:  body.Password,
		FirstName: body.FirstName,
		LastName:  body.LastName,
		Role:      body.Role,
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusCreated, resp)
}

// PATCH /v1/users/{id}/role
func (h *Handlers) UpdateUserRole(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Role == "" {
		respond.Error(w, http.StatusBadRequest, "role is required")
		return
	}

	resp, err := h.svc.Auth.UpdateUser(h.outgoingCtx(r), &authpb.UpdateUserRequest{
		UserId: chi.URLParam(r, "id"),
		Role:   body.Role,
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, resp)
}

// GET /v1/users
func (h *Handlers) ListUsers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	if page <= 0 {
		page = 1
	}
	resp, err := h.svc.Auth.ListUsers(h.outgoingCtx(r), &authpb.ListUsersRequest{
		Page:       int32(page),
		PageSize:   pageSize(r),
		Role:       q.Get("role"),
		BusinessId: q.Get("business_id"),
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, resp)
}

// PUT /v1/auth/password
func (h *Handlers) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var body struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
		ConfirmPassword string `json:"confirm_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.svc.Auth.ChangePassword(h.outgoingCtx(r), &authpb.ChangePasswordRequest{
		CurrentPassword: body.CurrentPassword,
		NewPassword:     body.NewPassword,
		ConfirmPassword: body.ConfirmPassword,
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, resp)
}
