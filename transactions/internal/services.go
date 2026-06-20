package internal

import (
	"context"
	"errors"

	
	"thugcorp.io/grocery/transaction/internal/domain"
)

type TransactionService interface {
	CreateTransaction(ctx context.Context, input CreateTransactionInput) (*domain.Transaction, error)
	GetTransaction(ctx context.Context, id string) (*domain.Transaction, error)
	UpdateTransactionStatus(ctx context.Context, id, status string) (*domain.Transaction, error)
	DeleteTransaction(ctx context.Context, id string) error
	ListTransactions(ctx context.Context, userID, businessID, status string, page, pageSize int) ([]*domain.Transaction, int64, error)
}

type CreateTransactionInput struct {
	UserID        string
	BusinessID    string
	PaymentMethod string
	Items         []TransactionItemInput
}

type TransactionItemInput struct {
	ProductID   string
	ProductName string
	BusinessID  string
	Quantity    int32
	Price       float64
}

type transactionService struct {
	repo      TransactionRepository
	
}

func NewTransactionService(
	repo TransactionRepository,
) TransactionService {
	return &transactionService{
		repo:      repo,
	}
}

func (s *transactionService) CreateTransaction(ctx context.Context, input CreateTransactionInput) (*domain.Transaction, error) {
	items := make([]domain.TransactionItem, 0, len(input.Items))
	var total float64
	for _, it := range input.Items {
		bid := it.BusinessID
		if bid == "" {
			bid = input.BusinessID
		}
		items = append(items, domain.TransactionItem{
			ProductID:   it.ProductID,
			ProductName: it.ProductName,
			BusinessID:  bid,
			Quantity:    it.Quantity,
			Price:       it.Price,
		})
		total += float64(it.Quantity) * it.Price
	}

	t := &domain.Transaction{
		UserID:        input.UserID,
		BusinessID:    input.BusinessID,
		PaymentMethod: input.PaymentMethod,
		TotalAmount:   total,
		Status:        "pending",
		Items:         items,
	}
	return s.repo.Create(ctx, t)
}

func (s *transactionService) GetTransaction(ctx context.Context, id string) (*domain.Transaction, error) {
	t, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, errors.New("transaction not found")
	}
	return t, nil
}

func (s *transactionService) UpdateTransactionStatus(ctx context.Context, id, status string) (*domain.Transaction, error) {
	t, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, errors.New("transaction not found")
	}
	return s.repo.UpdateStatus(ctx, id, status)
}

func (s *transactionService) DeleteTransaction(ctx context.Context, id string) error {
	t, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if t == nil {
		return errors.New("transaction not found")
	}
	return s.repo.Delete(ctx, id)
}

func (s *transactionService) ListTransactions(ctx context.Context, userID, businessID, status string, page, pageSize int) ([]*domain.Transaction, int64, error) {
	return s.repo.List(ctx, userID, businessID, status, page, pageSize)
}
