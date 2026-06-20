package handlers

import (
	"encoding/json"
	"net/http"

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
