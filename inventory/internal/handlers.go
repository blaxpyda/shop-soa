package internal

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"thugcorp.io/grocery/inventory/internal/domain"
	pb "thugcorp.io/grocery/inventory/proto"
)

type inventoryHandler struct {
	pb.UnimplementedInventoryServiceServer
	svc InventoryService
}

func NewInventoryHandler(svc InventoryService) *inventoryHandler {
	return &inventoryHandler{svc: svc}
}

func (h *inventoryHandler) CheckAvailability(ctx context.Context, req *pb.CheckAvailabilityRequest) (*pb.CheckAvailabilityResponse, error) {
	queries := make([]AvailabilityQuery, 0, len(req.Items))
	for _, item := range req.Items {
		queries = append(queries, AvailabilityQuery{
			BusinessID: item.BusinessId,
			ProductID:  item.ProductId,
			LocationID: item.LocationId,
			Quantity:   item.Quantity,
		})
	}

	results, err := h.svc.CheckAvailability(ctx, queries)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}

	pbResults := make([]*pb.AvailabilityResult, 0, len(results))
	for _, r := range results {
		pbResults = append(pbResults, &pb.AvailabilityResult{
			ProductId:  r.ProductID,
			LocationId: r.LocationID,
			Available:  r.Available,
			Sufficient: r.Sufficient,
			State:      pb.StockState(pb.StockState_value[r.State]),
		})
	}
	return &pb.CheckAvailabilityResponse{Results: pbResults}, nil
}

func (h *inventoryHandler) GetStock(ctx context.Context, req *pb.GetStockRequest) (*pb.StockItem, error) {
	item, err := h.svc.GetStock(ctx, req.BusinessId, req.ProductId, req.LocationId)
	if err != nil {
		if err.Error() == "stock item not found" {
			return nil, status.Errorf(codes.NotFound, "%v", err)
		}
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return mapStockToProto(item), nil
}

func (h *inventoryHandler) Reserve(ctx context.Context, req *pb.ReserveRequest) (*pb.ReserveResponse, error) {
	var ttl time.Duration
	if req.Ttl != nil {
		ttl = req.Ttl.AsDuration()
	}

	lines := make([]ReserveLineInput, 0, len(req.Items))
	for _, item := range req.Items {
		lines = append(lines, ReserveLineInput{
			BusinessID: item.BusinessId,
			ProductID:  item.ProductId,
			LocationID: item.LocationId,
			Quantity:   item.Quantity,
		})
	}

	result, err := h.svc.Reserve(ctx, ReserveInput{
		OrderID:        req.OrderId,
		Items:          lines,
		TTL:            ttl,
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}

	resp := &pb.ReserveResponse{
		ReservationId: result.ReservationID,
		Status:        pb.ReservationStatus(pb.ReservationStatus_value[result.Status]),
	}
	if !result.ExpiresAt.IsZero() {
		resp.ExpiresAt = timestamppb.New(result.ExpiresAt)
	}
	for _, sf := range result.Shortfalls {
		resp.Shortfalls = append(resp.Shortfalls, &pb.AvailabilityResult{
			ProductId:  sf.ProductID,
			LocationId: sf.LocationID,
			Available:  sf.Available,
			Sufficient: false,
			State:      pb.StockState(pb.StockState_value[sf.State]),
		})
	}
	return resp, nil
}

func (h *inventoryHandler) CommitReservation(ctx context.Context, req *pb.CommitReservationRequest) (*pb.CommitReservationResponse, error) {
	if err := h.svc.CommitReservation(ctx, req.ReservationId, req.IdempotencyKey); err != nil {
		if err.Error() == "reservation not found" {
			return nil, status.Errorf(codes.NotFound, "%v", err)
		}
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return &pb.CommitReservationResponse{Committed: true}, nil
}

func (h *inventoryHandler) ReleaseReservation(ctx context.Context, req *pb.ReleaseReservationRequest) (*pb.ReleaseReservationResponse, error) {
	if err := h.svc.ReleaseReservation(ctx, req.ReservationId, req.Reason); err != nil {
		if err.Error() == "reservation not found" {
			return nil, status.Errorf(codes.NotFound, "%v", err)
		}
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return &pb.ReleaseReservationResponse{Released: true}, nil
}

func (h *inventoryHandler) AdjustStock(ctx context.Context, req *pb.AdjustStockRequest) (*pb.StockItem, error) {
	input := AdjustStockInput{
		BusinessID:      req.BusinessId,
		ProductID:       req.ProductId,
		LocationID:      req.LocationId,
		ExpectedVersion: req.ExpectedVersion,
		IdempotencyKey:  req.IdempotencyKey,
	}
	switch change := req.Change.(type) {
	case *pb.AdjustStockRequest_Delta:
		input.Delta = &change.Delta
	case *pb.AdjustStockRequest_SetTo:
		input.SetTo = &change.SetTo
	default:
		return nil, status.Error(codes.InvalidArgument, "either delta or set_to must be provided")
	}

	item, err := h.svc.AdjustStock(ctx, input)
	if err != nil {
		if err.Error() == "version conflict: stock was modified concurrently" {
			return nil, status.Errorf(codes.Aborted, "%v", err)
		}
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return mapStockToProto(item), nil
}

func (h *inventoryHandler) ListStock(ctx context.Context, req *pb.ListStockRequest) (*pb.ListStockResponse, error) {
	items, nextCursor, err := h.svc.ListStock(ctx,
		req.BusinessId,
		req.LocationId,
		req.StateFilter.String(),
		int(req.PageSize),
		req.PageToken,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}

	pbItems := make([]*pb.StockItem, 0, len(items))
	for _, item := range items {
		pbItems = append(pbItems, mapStockToProto(item))
	}
	return &pb.ListStockResponse{Items: pbItems, NextPageToken: nextCursor}, nil
}

// ---- Helpers ----

func mapStockToProto(item *domain.StockItem) *pb.StockItem {
	return &pb.StockItem{
		BusinessId:        item.BusinessID,
		ProductId:         item.ProductID,
		LocationId:        item.LocationID,
		OnHand:            item.OnHand,
		Reserved:          item.Reserved,
		Available:         item.Available(),
		LowStockThreshold: item.LowStockThreshold,
		State:             pb.StockState(pb.StockState_value[item.State]),
		Version:           item.Version,
		UpdatedAt:         timestamppb.New(item.UpdatedAt),
	}
}
