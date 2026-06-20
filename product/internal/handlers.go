package internal

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"thugcorp.io/grocery/product/internal/domain"
	"thugcorp.io/grocery/product/internal/middleware"
	pb "thugcorp.io/grocery/product/proto"
)

type productHandler struct {
	pb.UnimplementedProductServiceServer
	svc ProductService
}

func NewProductHandler(svc ProductService) *productHandler {
	return &productHandler{svc: svc}
}

func (h *productHandler) CreateProduct(ctx context.Context, req *pb.CreateProductRequest) (*pb.Product, error) {
	businessID, ok := ctx.Value(middleware.BusinessIDKey).(string)
	if !ok || businessID == "" {
		return nil, status.Error(codes.Unauthenticated, "missing business ID in token")
	}

	p, err := h.svc.CreateProduct(ctx, CreateProductInput{
		BusinessID: businessID,
		Name:       req.Name,
		Category:   req.Category,
		Price:      req.Salesprice,
		Quantity:   float64(req.Stock),
		ImageURL:   req.ImageUrl,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return mapProductToProto(p), nil
}

func (h *productHandler) GetProduct(ctx context.Context, req *pb.GetProductRequest) (*pb.Product, error) {
	p, err := h.svc.GetProduct(ctx, req.ProductId)
	if err != nil {
		if err.Error() == "product not found" {
			return nil, status.Errorf(codes.NotFound, "%v", err)
		}
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return mapProductToProto(p), nil
}

func (h *productHandler) UpdateProduct(ctx context.Context, req *pb.UpdateProductRequest) (*pb.Product, error) {
	p, err := h.svc.UpdateProduct(ctx, UpdateProductInput{
		ID:       req.ProductId,
		Name:     req.Name,
		Category: req.Category,
		Price:    req.Price,
		Quantity: float64(req.Stock),
		ImageURL: req.ImageUrl,
	})
	if err != nil {
		if err.Error() == "product not found" {
			return nil, status.Errorf(codes.NotFound, "%v", err)
		}
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return mapProductToProto(p), nil
}

func (h *productHandler) DeleteProduct(ctx context.Context, req *pb.DeleteProductRequest) (*pb.DeleteResponse, error) {
	if err := h.svc.DeleteProduct(ctx, req.ProductId); err != nil {
		if err.Error() == "product not found" {
			return nil, status.Errorf(codes.NotFound, "%v", err)
		}
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return &pb.DeleteResponse{Success: true}, nil
}

func (h *productHandler) ListProducts(ctx context.Context, req *pb.ListProductsRequest) (*pb.ListProductsResponse, error) {
	businessID, ok := ctx.Value(middleware.BusinessIDKey).(string)
	if !ok || businessID == "" {
		return nil, status.Error(codes.Unauthenticated, "missing business ID in token")
	}

	products, total, err := h.svc.ListProducts(ctx, businessID, req.Category, int(req.Page), int(req.PageSize))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}

	pbProducts := make([]*pb.Product, 0, len(products))
	for _, p := range products {
		pbProducts = append(pbProducts, mapProductToProto(p))
	}
	return &pb.ListProductsResponse{Products: pbProducts, Total: int32(total)}, nil
}

func (h *productHandler) SearchProducts(ctx context.Context, req *pb.SearchProductsRequest) (*pb.ListProductsResponse, error) {
	businessID, ok := ctx.Value(middleware.BusinessIDKey).(string)
	if !ok || businessID == "" {
		return nil, status.Error(codes.Unauthenticated, "missing business ID in token")
	}

	products, total, err := h.svc.SearchProducts(ctx, businessID, req.Query, int(req.Page), int(req.PageSize))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}

	pbProducts := make([]*pb.Product, 0, len(products))
	for _, p := range products {
		pbProducts = append(pbProducts, mapProductToProto(p))
	}
	return &pb.ListProductsResponse{Products: pbProducts, Total: int32(total)}, nil
}

func mapProductToProto(p *domain.Product) *pb.Product {
	return &pb.Product{
		Id:         p.ID,
		BusinessId: p.BusinessID,
		Name:       p.Name,
		Category:   p.Category,
		Salesprice: p.Price,
		Stock:      int32(p.Quantity),
		ImageUrl:   p.ImageURL,
		IsActive:   p.IsActive,
	}
}
