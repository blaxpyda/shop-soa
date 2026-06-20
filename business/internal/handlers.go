package internal

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"thugcorp.io/grocery/business/internal/domain"
	"thugcorp.io/grocery/business/internal/middleware"
	pb "thugcorp.io/grocery/business/proto"
)

type businessHandler struct {
	pb.UnimplementedBusinessServiceServer
	svc BusinessService
}

func NewBusinessHandler(svc BusinessService) *businessHandler {
	return &businessHandler{svc: svc}
}

func (h *businessHandler) CreateBusiness(ctx context.Context, req *pb.CreateBusinessRequest) (*pb.Business, error) {
	callerID, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	input := domain.CreateBusinessInput{
		Name:        req.Name,
		Description: req.Description,
		Email:       req.Email,
		Phone:       req.Phone,
		Address:     req.Address,
		City:        req.City,
		Country:     req.Country,
	}

	b, err := h.svc.CreateBusiness(ctx, callerID, input)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}
	return mapToProto(b), nil
}

func (h *businessHandler) GetBusiness(ctx context.Context, req *pb.GetBusinessRequest) (*pb.Business, error) {
	b, err := h.svc.GetBusiness(ctx, req.BusinessId)
	if err != nil {
		if err.Error() == "business not found" {
			return nil, status.Errorf(codes.NotFound, "%v", err)
		}
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return mapToProto(b), nil
}

func (h *businessHandler) UpdateBusiness(ctx context.Context, req *pb.UpdateBusinessRequest) (*pb.Business, error) {
	callerID, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	callerRole, _ := ctx.Value(middleware.RoleKey).(string)

	input := domain.UpdateBusinessInput{
		Name:        req.Name,
		Description: req.Description,
		Email:       req.Email,
		Phone:       req.Phone,
		Address:     req.Address,
		City:        req.City,
		Country:     req.Country,
		IsActive:    &req.IsActive,
	}

	b, err := h.svc.UpdateBusiness(ctx, callerID, callerRole, req.BusinessId, input)
	if err != nil {
		switch err.Error() {
		case "business not found":
			return nil, status.Errorf(codes.NotFound, "%v", err)
		case "forbidden: only the owner or an admin can update this business":
			return nil, status.Errorf(codes.PermissionDenied, "%v", err)
		default:
			return nil, status.Errorf(codes.Internal, "%v", err)
		}
	}
	return mapToProto(b), nil
}

func (h *businessHandler) DeleteBusiness(ctx context.Context, req *pb.DeleteBusinessRequest) (*pb.DeleteBusinessResponse, error) {
	callerID, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	callerRole, _ := ctx.Value(middleware.RoleKey).(string)

	if err := h.svc.DeleteBusiness(ctx, callerID, callerRole, req.BusinessId); err != nil {
		switch err.Error() {
		case "business not found":
			return nil, status.Errorf(codes.NotFound, "%v", err)
		case "forbidden: only the owner or a super-admin can delete this business":
			return nil, status.Errorf(codes.PermissionDenied, "%v", err)
		default:
			return nil, status.Errorf(codes.Internal, "%v", err)
		}
	}
	return &pb.DeleteBusinessResponse{Success: true}, nil
}

// ---- Helpers ----

func userIDFromCtx(ctx context.Context) (string, error) {
	userID, ok := ctx.Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		return "", status.Errorf(codes.Unauthenticated, "user ID not found in context")
	}
	return userID, nil
}

func mapToProto(b *domain.Business) *pb.Business {
	return &pb.Business{
		Id:          b.ID,
		OwnerId:     b.OwnerID,
		Name:        b.Name,
		Description: b.Description,
		Email:       b.Email,
		Phone:       b.Phone,
		Address:     b.Address,
		City:        b.City,
		Country:     b.Country,
		IsActive:    b.IsActive,
		CreatedAt:   b.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   b.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
