package internal

import (
	"context"
	"strconv"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"thugcorp.io/catalog/internal/domain"
	"thugcorp.io/catalog/internal/middleware"
	pb "thugcorp.io/catalog/proto"
)

type catalogHandler struct {
	pb.UnimplementedCatalogServiceServer
	svc CatalogService
}

func NewCatalogHandler(svc CatalogService) *catalogHandler {
	return &catalogHandler{svc: svc}
}

// ---- Products ----

func (h *catalogHandler) CreateProduct(ctx context.Context, req *pb.CreateProductRequest) (*pb.Product, error) {
	businessID := businessIDFromCtx(ctx)
	if businessID == "" {
		businessID = req.BusinessId
	}

	input := domain.CreateProductInput{
		Name:         req.Name,
		Description:  req.Description,
		Category:     req.Category,
		Price:        req.Price,
		Currency:     req.Currency,
		ImageURL:     req.ImageUrl,
		InitialStock: req.InitialStock,
	}

	product, err := h.svc.CreateProduct(ctx, businessID, input)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}
	return mapProduct(product), nil
}

func (h *catalogHandler) GetProduct(ctx context.Context, req *pb.GetProductRequest) (*pb.Product, error) {
	product, err := h.svc.GetProduct(ctx, req.ProductId)
	if err != nil {
		if err.Error() == "product not found" {
			return nil, status.Errorf(codes.NotFound, "%v", err)
		}
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return mapProduct(product), nil
}

func (h *catalogHandler) UpdateProduct(ctx context.Context, req *pb.UpdateProductRequest) (*pb.Product, error) {
	businessID := businessIDFromCtx(ctx)
	active := req.Active
	input := domain.UpdateProductInput{
		Name:        req.Name,
		Description: req.Description,
		Category:    req.Category,
		Price:       req.Price,
		Active:      &active,
	}

	product, err := h.svc.UpdateProduct(ctx, req.ProductId, businessID, input)
	if err != nil {
		switch err.Error() {
		case "product not found":
			return nil, status.Errorf(codes.NotFound, "%v", err)
		case "unauthorized: product does not belong to your business":
			return nil, status.Errorf(codes.PermissionDenied, "%v", err)
		}
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return mapProduct(product), nil
}

func (h *catalogHandler) ListProducts(ctx context.Context, req *pb.ListProductsRequest) (*pb.ListProductsResponse, error) {
	pageSize := int(req.PageSize)
	filter := domain.ListProductsFilter{
		BusinessID: req.BusinessId,
		PageSize:   pageSize,
		PageToken:  req.PageToken,
	}

	products, err := h.svc.ListProducts(ctx, filter)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}

	resp := &pb.ListProductsResponse{}
	for _, p := range products {
		resp.Products = append(resp.Products, mapProduct(p))
	}

	// Advance cursor when a full page was returned.
	if pageSize > 0 && len(products) == pageSize {
		offset := 0
		if req.PageToken != "" {
			if v, err := strconv.Atoi(req.PageToken); err == nil {
				offset = v
			}
		}
		resp.NextPageToken = strconv.Itoa(offset + pageSize)
	}

	return resp, nil
}

// ---- Inventory ----

func (h *catalogHandler) CheckAvailability(ctx context.Context, req *pb.CheckAvailabilityRequest) (*pb.CheckAvailabilityResponse, error) {
	queries := make([]domain.AvailabilityQuery, len(req.Items))
	for i, item := range req.Items {
		queries[i] = domain.AvailabilityQuery{
			ProductID:  item.ProductId,
			LocationID: item.LocationId,
			Quantity:   item.Quantity,
		}
	}

	results, err := h.svc.CheckAvailability(ctx, queries)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}

	resp := &pb.CheckAvailabilityResponse{}
	for _, r := range results {
		resp.Results = append(resp.Results, &pb.AvailabilityResult{
			ProductId:  r.ProductID,
			Available:  r.Available,
			Sufficient: r.Sufficient,
		})
	}
	return resp, nil
}

func (h *catalogHandler) AdjustStock(ctx context.Context, req *pb.AdjustStockRequest) (*pb.StockItem, error) {
	input := domain.AdjustStockInput{
		ProductID:       req.ProductId,
		LocationID:      req.LocationId,
		Reason:          req.Reason.String(),
		ExpectedVersion: req.ExpectedVersion,
		IdempotencyKey:  req.IdempotencyKey,
	}

	switch c := req.Change.(type) {
	case *pb.AdjustStockRequest_Delta:
		delta := c.Delta
		input.Delta = &delta
	case *pb.AdjustStockRequest_SetTo:
		setTo := c.SetTo
		input.SetTo = &setTo
	default:
		return nil, status.Errorf(codes.InvalidArgument, "either delta or set_to must be specified")
	}

	stock, err := h.svc.AdjustStock(ctx, input)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return mapStockItem(stock), nil
}

// ---- Checkout saga ----

func (h *catalogHandler) Reserve(ctx context.Context, req *pb.ReserveRequest) (*pb.ReserveResponse, error) {
	ttl := 10 * time.Minute
	if req.Ttl != nil {
		ttl = req.Ttl.AsDuration()
	}

	items := make([]domain.ReserveItemInput, len(req.Items))
	for i, item := range req.Items {
		items[i] = domain.ReserveItemInput{
			ProductID:  item.ProductId,
			LocationID: item.LocationId,
			Quantity:   item.Quantity,
		}
	}

	reservation, shortfalls, held, err := h.svc.Reserve(ctx, domain.ReserveInput{
		OrderID:        req.OrderId,
		Items:          items,
		TTL:            ttl,
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}

	resp := &pb.ReserveResponse{Held: held}
	if held && reservation != nil {
		resp.ReservationId = reservation.ID
		resp.ExpiresAt = timestamppb.New(reservation.ExpiresAt)
	} else {
		for _, sf := range shortfalls {
			resp.Shortfalls = append(resp.Shortfalls, &pb.AvailabilityResult{
				ProductId:  sf.ProductID,
				Available:  sf.Available,
				Sufficient: false,
			})
		}
	}
	return resp, nil
}

func (h *catalogHandler) CommitReservation(ctx context.Context, req *pb.CommitReservationRequest) (*pb.CommitReservationResponse, error) {
	if err := h.svc.CommitReservation(ctx, req.ReservationId); err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return &pb.CommitReservationResponse{Committed: true}, nil
}

func (h *catalogHandler) ReleaseReservation(ctx context.Context, req *pb.ReleaseReservationRequest) (*pb.ReleaseReservationResponse, error) {
	if err := h.svc.ReleaseReservation(ctx, req.ReservationId); err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return &pb.ReleaseReservationResponse{Released: true}, nil
}

// ---- Helpers ----

func businessIDFromCtx(ctx context.Context) string {
	bid, _ := ctx.Value(middleware.BusinessIDKey).(string)
	return bid
}

func mapProduct(p *domain.Product) *pb.Product {
	return &pb.Product{
		Id:          p.ID,
		BusinessId:  p.BusinessID,
		Name:        p.Name,
		Description: p.Description,
		Category:    p.Category,
		Price:       p.Price,
		Currency:    p.Currency,
		ImageUrl:    p.ImageURL,
		Active:      p.Active,
		CreatedAt:   timestamppb.New(p.CreatedAt),
	}
}

func mapStockItem(s *domain.StockItem) *pb.StockItem {
	return &pb.StockItem{
		ProductId:  s.ProductID,
		BusinessId: s.BusinessID,
		LocationId: s.LocationID,
		OnHand:     s.OnHand,
		Reserved:   s.Reserved,
		Available:  s.Available(),
		Version:    s.Version,
	}
}
