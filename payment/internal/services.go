package internal

import (
	"fmt"
	"time"

	"thugcorp.io/payment/internal/domain"
)

type PaymentService interface {
	InitiatePayment(input domain.InitiatePaymentInput) (*domain.Payment, error)
	GetPayment(id string) (*domain.Payment, error)
	Refund(input domain.RefundInput) (*domain.Payment, error)
	HandleWebhook(paymentID string, success bool, providerRef string) error

	ListTransactions(filter domain.ListTransactionsFilter) ([]domain.Transaction, string, error)
	GetBalance(businessID string) (int64, string, error)
	InitiatePayout(input domain.InitiatePayoutInput) (*domain.Payout, error)
}

type paymentService struct {
	repo PaymentRepository
}

func NewPaymentService(repo PaymentRepository) PaymentService {
	return &paymentService{repo: repo}
}

func (s *paymentService) InitiatePayment(input domain.InitiatePaymentInput) (*domain.Payment, error) {
	if input.OrderID == "" {
		return nil, fmt.Errorf("order_id is required")
	}
	if input.UserID == "" {
		return nil, fmt.Errorf("user_id is required")
	}
	if input.Amount <= 0 {
		return nil, fmt.Errorf("amount must be positive")
	}
	if input.Currency == "" {
		return nil, fmt.Errorf("currency is required")
	}
	if input.Phone == "" {
		return nil, fmt.Errorf("phone is required")
	}
	if input.IdempotencyKey == "" {
		return nil, fmt.Errorf("idempotency_key is required")
	}

	if existing, err := s.repo.GetPaymentByIdempotencyKey(input.IdempotencyKey); err != nil {
		return nil, err
	} else if existing != nil {
		return existing, nil
	}

	p := &domain.Payment{
		ID:             newID(),
		OrderID:        input.OrderID,
		UserID:         input.UserID,
		Amount:         input.Amount,
		Currency:       input.Currency,
		Provider:       input.Provider,
		Phone:          input.Phone,
		Status:         domain.PaymentStatusPending,
		IdempotencyKey: input.IdempotencyKey,
	}
	if err := s.repo.CreatePayment(p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *paymentService) GetPayment(id string) (*domain.Payment, error) {
	return s.repo.GetPaymentByID(id)
}

func (s *paymentService) Refund(input domain.RefundInput) (*domain.Payment, error) {
	if input.PaymentID == "" {
		return nil, fmt.Errorf("payment_id is required")
	}
	if input.IdempotencyKey == "" {
		return nil, fmt.Errorf("idempotency_key is required")
	}

	payment, err := s.repo.GetPaymentByID(input.PaymentID)
	if err != nil {
		return nil, err
	}
	if payment.Status != domain.PaymentStatusSucceeded {
		return nil, fmt.Errorf("can only refund succeeded payments, current status: %s", payment.Status)
	}

	refundAmount := input.Amount
	if refundAmount == 0 {
		refundAmount = payment.Amount
	}
	if refundAmount > payment.Amount {
		return nil, fmt.Errorf("refund amount exceeds payment amount")
	}

	if err := s.repo.UpdatePaymentStatus(payment.ID, domain.PaymentStatusRefunded, payment.ProviderRef); err != nil {
		return nil, err
	}

	txn := &domain.Transaction{
		ID:         newID(),
		PaymentID:  payment.ID,
		BusinessID: "",
		Type:       domain.TransactionTypeRefund,
		Amount:     refundAmount,
		Currency:   payment.Currency,
		Note:       input.Reason,
	}
	if err := s.repo.CreateTransaction(txn); err != nil {
		return nil, err
	}

	payment.Status = domain.PaymentStatusRefunded
	return payment, nil
}

// HandleWebhook is called by the HTTP webhook endpoint when a provider confirms.
func (s *paymentService) HandleWebhook(paymentID string, success bool, providerRef string) error {
	payment, err := s.repo.GetPaymentByID(paymentID)
	if err != nil {
		return err
	}
	if payment.Status != domain.PaymentStatusPending {
		return nil // already processed, idempotent
	}

	if !success {
		return s.repo.UpdatePaymentStatus(paymentID, domain.PaymentStatusFailed, providerRef)
	}

	if err := s.repo.UpdatePaymentStatus(paymentID, domain.PaymentStatusSucceeded, providerRef); err != nil {
		return err
	}

	// Write payment ledger entry for the business (no commission logic here — extend as needed)
	txn := &domain.Transaction{
		ID:        newID(),
		PaymentID: payment.ID,
		Type:      domain.TransactionTypePayment,
		Amount:    payment.Amount,
		Currency:  payment.Currency,
		Note:      fmt.Sprintf("payment for order %s", payment.OrderID),
		CreatedAt: time.Now(),
	}
	return s.repo.CreateTransaction(txn)
}

func (s *paymentService) ListTransactions(filter domain.ListTransactionsFilter) ([]domain.Transaction, string, error) {
	return s.repo.ListTransactions(filter)
}

func (s *paymentService) GetBalance(businessID string) (int64, string, error) {
	if businessID == "" {
		return 0, "", fmt.Errorf("business_id is required")
	}
	sums, currency, err := s.repo.SumTransactionsByBusiness(businessID)
	if err != nil {
		return 0, "", err
	}

	balance := sums[domain.TransactionTypePayment] -
		sums[domain.TransactionTypeCommission] -
		sums[domain.TransactionTypePayout]
	return balance, currency, nil
}

func (s *paymentService) InitiatePayout(input domain.InitiatePayoutInput) (*domain.Payout, error) {
	if input.BusinessID == "" {
		return nil, fmt.Errorf("business_id is required")
	}
	if input.Amount <= 0 {
		return nil, fmt.Errorf("amount must be positive")
	}
	if input.Currency == "" {
		return nil, fmt.Errorf("currency is required")
	}
	if input.Phone == "" {
		return nil, fmt.Errorf("phone is required")
	}
	if input.IdempotencyKey == "" {
		return nil, fmt.Errorf("idempotency_key is required")
	}

	if existing, err := s.repo.GetPayoutByIdempotencyKey(input.IdempotencyKey); err != nil {
		return nil, err
	} else if existing != nil {
		return existing, nil
	}

	balance, _, err := s.GetBalance(input.BusinessID)
	if err != nil {
		return nil, err
	}
	if balance < input.Amount {
		return nil, fmt.Errorf("insufficient balance: have %d, requested %d", balance, input.Amount)
	}

	payout := &domain.Payout{
		ID:             newID(),
		BusinessID:     input.BusinessID,
		Amount:         input.Amount,
		Currency:       input.Currency,
		Provider:       input.Provider,
		Phone:          input.Phone,
		Status:         domain.PaymentStatusPending,
		IdempotencyKey: input.IdempotencyKey,
	}
	if err := s.repo.CreatePayout(payout); err != nil {
		return nil, err
	}

	// Record the debit immediately so balance is reserved
	txn := &domain.Transaction{
		ID:         newID(),
		BusinessID: input.BusinessID,
		Type:       domain.TransactionTypePayout,
		Amount:     input.Amount,
		Currency:   input.Currency,
		Note:       fmt.Sprintf("payout %s", payout.ID),
	}
	if err := s.repo.CreateTransaction(txn); err != nil {
		return nil, err
	}
	return payout, nil
}
