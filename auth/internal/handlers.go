package internal

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"thugcorp.io/grocery/auth/internal/domain"
	"thugcorp.io/grocery/auth/internal/middleware"
	pb "thugcorp.io/grocery/auth/proto"
)

type authHandler struct {
	pb.UnimplementedIdentityServiceServer
	authService AuthService
}

func NewAuthHandler(authService AuthService) *authHandler {
	return &authHandler{authService: authService}
}

// ---- Public flows ----

func (h *authHandler) Signup(ctx context.Context, req *pb.SignupRequest) (*pb.AuthResponse, error) {
	input := domain.CreateUserInput{
		Email:    req.Email,
		Phone:    req.Phone,
		Password: req.Password,
	}

	user, needsVerify, verifyMethod, err := h.authService.Signup(ctx, input)
	if err != nil {
		return nil, status.Errorf(codes.AlreadyExists, "%v", err)
	}

	resp := &pb.AuthResponse{
		UserId:      user.ID,
		NeedsVerify: needsVerify,
	}
	if needsVerify {
		resp.VerifyMethod = verifyMethod
	}
	return resp, nil
}

func (h *authHandler) Login(ctx context.Context, req *pb.LoginRequest) (*pb.AuthResponse, error) {
	user, token, err := h.authService.Login(ctx, req.Email, req.Phone, req.Password)
	if err != nil {
		if err.Error() == "account not verified" {
			verifyMethod := "email"
			if user.Email == "" {
				verifyMethod = "sms"
			}
			return &pb.AuthResponse{
				UserId:       user.ID,
				NeedsVerify:  true,
				VerifyMethod: verifyMethod,
			}, nil
		}
		return nil, status.Errorf(codes.Unauthenticated, "%v", err)
	}

	return &pb.AuthResponse{
		UserId:     user.ID,
		Token:      token,
		Role:       user.Role,
		BusinessId: user.BusinessID,
	}, nil
}

func (h *authHandler) VerifyCode(ctx context.Context, req *pb.VerifyCodeRequest) (*pb.AuthResponse, error) {
	user, token, err := h.authService.VerifyCode(ctx, req.UserId, req.Code)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}

	return &pb.AuthResponse{
		UserId:     user.ID,
		Token:      token,
		Role:       user.Role,
		BusinessId: user.BusinessID,
	}, nil
}

func (h *authHandler) ResendCode(ctx context.Context, req *pb.ResendCodeRequest) (*pb.EmptyResponse, error) {
	if err := h.authService.ResendCode(ctx, req.UserId); err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return &pb.EmptyResponse{Success: true}, nil
}

// ---- Self-service (JWT required, user acts on own account) ----

func (h *authHandler) GetProfile(ctx context.Context, req *pb.GetProfileRequest) (*pb.User, error) {
	userID, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	user, err := h.authService.GetProfile(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get profile: %v", err)
	}

	return mapToProfileResponse(user), nil
}

func (h *authHandler) UpdateProfile(ctx context.Context, req *pb.UpdateProfileRequest) (*pb.User, error) {
	userID, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	input := domain.UpdateUserInput{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     req.Email,
		Phone:     req.Phone,
		Address:   req.Address,
		City:      req.City,
		Country:   req.Country,
	}

	user, err := h.authService.UpdateProfile(ctx, userID, input)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update profile: %v", err)
	}

	return mapToProfileResponse(user), nil
}

func (h *authHandler) ChangePassword(ctx context.Context, req *pb.ChangePasswordRequest) (*pb.EmptyResponse, error) {
	userID, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	if req.NewPassword != req.ConfirmPassword {
		return nil, status.Errorf(codes.InvalidArgument, "new password and confirmation do not match")
	}

	if err := h.authService.ChangePassword(ctx, userID, req.CurrentPassword, req.NewPassword); err != nil {
		if err.Error() == "current password is incorrect" {
			return nil, status.Errorf(codes.InvalidArgument, "%v", err)
		}
		return nil, status.Errorf(codes.Internal, "failed to change password: %v", err)
	}

	return &pb.EmptyResponse{Success: true}, nil
}

func (h *authHandler) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.User, error) {
	callerRole, _ := ctx.Value(middleware.RoleKey).(string)
	input := domain.CreateUserInput{
		Email:      req.Email,
		Phone:      req.Phone,
		Password:   req.Password,
		FirstName:  req.FirstName,
		LastName:   req.LastName,
		Role:       req.Role,
		BusinessID: req.BusinessId,
	}
	user, err := h.authService.CreateUser(ctx, callerRole, input)
	if err != nil {
		switch err.Error() {
		case "only super-admin can create users", "admins cannot create users with elevated roles":
			return nil, status.Errorf(codes.PermissionDenied, "%v", err)
		case "business_id is required":
			return nil, status.Errorf(codes.InvalidArgument, "%v", err)
		default:
			return nil, status.Errorf(codes.Internal, "failed to create user: %v", err)
		}
	}
	return mapToProfileResponse(user), nil
}

func (h *authHandler) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.User, error) {
	callerRole, _ := ctx.Value(middleware.RoleKey).(string)
	callerID, _ := ctx.Value(middleware.UserIDKey).(string)
	input := domain.UpdateUserInput{
		FirstName:  req.FirstName,
		LastName:   req.LastName,
		Email:      req.Email,
		Phone:      req.Phone,
		Role:       req.Role,
		BusinessID: req.BusinessId,
	}
	user, err := h.authService.UpdateUser(ctx, callerRole, callerID, req.UserId, input)
	if err != nil {
		if err.Error() == "only super-admin can update users" {
			return nil, status.Errorf(codes.PermissionDenied, "%v", err)
		}
		return nil, status.Errorf(codes.Internal, "failed to update user: %v", err)
	}
	return mapToProfileResponse(user), nil
}

func (h *authHandler) ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
	filter := domain.ListUsersFilter{
		Page:       int(req.Page),
		PageSize:   int(req.PageSize),
		Role:       req.Role,
		BusinessID: req.BusinessId,
	}

	users, err := h.authService.ListUsers(ctx, filter)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list users: %v", err)
	}

	resp := &pb.ListUsersResponse{}
	for _, user := range users {
		resp.Users = append(resp.Users, mapToProfileResponse(user))
	}
	return resp, nil
}

// ---- Helpers ----

func userIDFromCtx(ctx context.Context) (string, error) {
	userID, ok := ctx.Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		return "", status.Errorf(codes.Unauthenticated, "user ID not found in context")
	}
	return userID, nil
}

func mapToProfileResponse(user *domain.User) *pb.User {
	return &pb.User{
		Id:        user.ID,
		Email:     user.Email,
		Phone:     user.Phone,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Address:   user.Address,
		Country:   user.Country,
		Role:      user.Role,
	}
}
