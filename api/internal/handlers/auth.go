package handlers

import (
	"encoding/json"
	"net/http"

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

// GET /v1/admin/users
func (h *Handlers) ListUsers(w http.ResponseWriter, r *http.Request) {
	resp, err := h.svc.Auth.ListUsers(h.outgoingCtx(r), &authpb.ListUsersRequest{
		Page:     1,
		PageSize: pageSize(r),
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}

	type userRow struct {
		ID         string `json:"id"`
		FullName   string `json:"full_name"`
		Email      string `json:"email"`
		Phone      string `json:"phone"`
		Role       string `json:"role"`
		Status     string `json:"status"`
		BusinessID string `json:"business_id,omitempty"`
	}
	type listResp struct {
		Users []userRow `json:"users"`
	}

	out := listResp{Users: make([]userRow, 0, len(resp.Users))}
	for _, u := range resp.Users {
		out.Users = append(out.Users, userRow{
			ID:         u.Id,
			FullName:   u.FirstName + " " + u.LastName,
			Email:      u.Email,
			Phone:      u.Phone,
			Role:       u.Role,
			Status:     "active",
			BusinessID: u.BusinessId,
		})
	}
	respond.JSON(w, http.StatusOK, out)
}

// POST /v1/admin/user
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

	u, err := h.svc.Auth.CreateUser(h.outgoingCtx(r), &authpb.CreateUserRequest{
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
	respond.JSON(w, http.StatusCreated, u)
}

// PATCH /v1/admin/users/{userId}/role
func (h *Handlers) UpdateUserRole(w http.ResponseWriter, r *http.Request) {
	userId := chi.URLParam(r, "userId")
	if userId == "" {
		respond.Error(w, http.StatusBadRequest, "userId is required")
		return
	}

	var body struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	u, err := h.svc.Auth.UpdateUser(h.outgoingCtx(r), &authpb.UpdateUserRequest{
		UserId: userId,
		Role:   body.Role,
	})
	if err != nil {
		respond.GRPCError(w, err)
		return
	}
	respond.JSON(w, http.StatusOK, u)
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
