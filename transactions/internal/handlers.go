package internal

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"thugcorp.io/grocery/transaction/internal/domain"
	pb "thugcorp.io/grocery/transaction/proto"
)

type transactionHandler struct {
	pb.UnimplementedTransactionServiceServer
	svc TransactionService
}

func NewTransactionHandler(svc TransactionService) *transactionHandler {
	return &transactionHandler{svc: svc}
}

func (h *transactionHandler) CreateTransaction(ctx context.Context, req *pb.CreateTransactionRequest) (*pb.Transaction, error) {
	items := make([]TransactionItemInput, 0, len(req.Items))
	for _, it := range req.Items {
		items = append(items, TransactionItemInput{
			ProductID:   it.ProductId,
			ProductName: it.ProductName,
			BusinessID:  it.BusinessId,
			Quantity:    it.Quantity,
			Price:       it.Price,
		})
	}

	t, err := h.svc.CreateTransaction(ctx, CreateTransactionInput{
		UserID:        req.UserId,
		BusinessID:    req.BusinessId,
		PaymentMethod: req.PaymentMethod,
		Items:         items,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return mapTransactionToProto(t), nil
}

func (h *transactionHandler) GetTransaction(ctx context.Context, req *pb.GetTransactionRequest) (*pb.Transaction, error) {
	t, err := h.svc.GetTransaction(ctx, req.TransactionId)
	if err != nil {
		if err.Error() == "transaction not found" {
			return nil, status.Errorf(codes.NotFound, "%v", err)
		}
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return mapTransactionToProto(t), nil
}

func (h *transactionHandler) UpdateTransactionStatus(ctx context.Context, req *pb.UpdateStatusRequest) (*pb.Transaction, error) {
	t, err := h.svc.UpdateTransactionStatus(ctx, req.TransactionId, req.Status)
	if err != nil {
		if err.Error() == "transaction not found" {
			return nil, status.Errorf(codes.NotFound, "%v", err)
		}
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return mapTransactionToProto(t), nil
}

func (h *transactionHandler) DeleteTransaction(ctx context.Context, req *pb.DeleteTransactionRequest) (*pb.DeleteResponse, error) {
	if err := h.svc.DeleteTransaction(ctx, req.TransactionId); err != nil {
		if err.Error() == "transaction not found" {
			return nil, status.Errorf(codes.NotFound, "%v", err)
		}
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return &pb.DeleteResponse{Success: true}, nil
}

func (h *transactionHandler) ListTransactions(ctx context.Context, req *pb.ListTransactionsRequest) (*pb.ListTransactionsResponse, error) {
	transactions, total, err := h.svc.ListTransactions(ctx,
		req.UserId, req.BusinessId, req.Status,
		int(req.Page), int(req.PageSize),
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}

	pbTxns := make([]*pb.Transaction, 0, len(transactions))
	for _, t := range transactions {
		pbTxns = append(pbTxns, mapTransactionToProto(t))
	}
	return &pb.ListTransactionsResponse{Transactions: pbTxns, Total: int32(total)}, nil
}

func mapTransactionToProto(t *domain.Transaction) *pb.Transaction {
	items := make([]*pb.TransactionItem, 0, len(t.Items))
	for _, it := range t.Items {
		items = append(items, &pb.TransactionItem{
			ProductId:   it.ProductID,
			ProductName: it.ProductName,
			BusinessId:  it.BusinessID,
			Quantity:    it.Quantity,
			Price:       it.Price,
		})
	}
	return &pb.Transaction{
		Id:            t.ID,
		UserId:        t.UserID,
		BusinessId:    t.BusinessID,
		Items:         items,
		TotalAmount:   t.TotalAmount,
		Status:        t.Status,
		PaymentMethod: t.PaymentMethod,
	}
}
