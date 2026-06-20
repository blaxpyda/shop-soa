package internal

import (
	"context"
	"fmt"
	"strconv"

	"thugcorp.io/grocery/payment/internal/domain"
	"thugcorp.io/grocery/payment/internal/mtn"
)

const mtnMSISDN = "MSISDN"

type PaymentService interface {
	InitiatePayment(ctx context.Context, input domain.CreatePaymentInput) (*domain.Payment, error)
	ConfirmPayment(ctx context.Context, paymentID, transactionRef, provider string) (*domain.Payment, error)
	GetPaymentStatus(ctx context.Context, paymentID string) (*domain.Payment, error)
	RefundPayment(ctx context.Context, paymentID string, amount float64, reason string) (*domain.Payment, error)
	ListPayments(ctx context.Context, filter domain.ListFilter) ([]*domain.Payment, int64, error)
}

type paymentService struct {
	repo PaymentRepository
	mtn  *mtn.Client
}

func NewPaymentService(repo PaymentRepository, mtnClient *mtn.Client) PaymentService {
	return &paymentService{repo: repo, mtn: mtnClient}
}

func (s *paymentService) InitiatePayment(ctx context.Context, input domain.CreatePaymentInput) (*domain.Payment, error) {
	if input.OrderID == "" {
		return nil, fmt.Errorf("%w: order ID is required", domain.ErrInvalidInput)
	}
	if input.Amount <= 0 {
		return nil, fmt.Errorf("%w: amount must be greater than zero", domain.ErrInvalidInput)
	}
	if input.Currency == "" {
		return nil, fmt.Errorf("%w: currency is required", domain.ErrInvalidInput)
	}

	if input.Provider == "mtn" {
		return s.payByMtn(ctx, input)
	}
	return s.repo.Create(ctx, input)
}

func (s *paymentService) ConfirmPayment(ctx context.Context, paymentID, transactionRef, provider string) (*domain.Payment, error) {
	p, err := s.repo.GetByID(ctx, paymentID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, domain.ErrNotFound
	}
	if p.Status != domain.StatusPending {
		return nil, fmt.Errorf("%w: payment is not pending", domain.ErrInvalidState)
	}

	status := domain.StatusCompleted
	return s.repo.Update(ctx, paymentID, domain.UpdatePaymentInput{
		Status:         &status,
		TransactionRef: &transactionRef,
	})
}

func (s *paymentService) GetPaymentStatus(ctx context.Context, paymentID string) (*domain.Payment, error) {
	p, err := s.repo.GetByID(ctx, paymentID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, domain.ErrNotFound
	}
	return p, nil
}

func (s *paymentService) RefundPayment(ctx context.Context, paymentID string, amount float64, reason string) (*domain.Payment, error) {
	p, err := s.repo.GetByID(ctx, paymentID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, domain.ErrNotFound
	}
	if p.Status != domain.StatusCompleted {
		return nil, fmt.Errorf("%w: only completed payments can be refunded", domain.ErrInvalidState)
	}

	status := domain.StatusRefunded
	return s.repo.Update(ctx, paymentID, domain.UpdatePaymentInput{
		Status:       &status,
		ErrorMessage: &reason,
	})
}

func (s *paymentService) ListPayments(ctx context.Context, filter domain.ListFilter) ([]*domain.Payment, int64, error) {
	return s.repo.List(ctx, filter)
}

func (s *paymentService) payByMtn(ctx context.Context, input domain.CreatePaymentInput) (*domain.Payment, error) {
	if s.mtn == nil {
		return nil, fmt.Errorf("%w: MTN client not configured", domain.ErrInvalidInput)
	}
	payerPhone, ok := input.Metadata["payer_phone"]
	if !ok || payerPhone == "" {
		return nil, fmt.Errorf("%w: payer_phone is required for MTN payments", domain.ErrInvalidInput)
	}
	payeePhone, ok := input.Metadata["payee_phone"]
	if !ok || payeePhone == "" {
		return nil, fmt.Errorf("%w: payee_phone is required for MTN payments", domain.ErrInvalidInput)
	}
	validityDuration := input.Metadata["validity_duration"]
	if validityDuration == "" {
		validityDuration = "3600" // default 1 hour
	}

	p, err := s.repo.Create(ctx, input)
	if err != nil {
		return nil, err
	}

	refID, err := s.mtn.CreateInvoice(ctx, mtn.CreateInvoiceRequest{
		ExternalID:       p.ID,
		Amount:           strconv.FormatFloat(input.Amount, 'f', -1, 64),
		Currency:         input.Currency,
		ValidityDuration: validityDuration,
		IntendedPayer:    mtn.Party{PartyIDType: mtnMSISDN, PartyID: payerPhone},
		Payee:            mtn.Party{PartyIDType: mtnMSISDN, PartyID: payeePhone},
		Description:      input.Metadata["description"],
	})
	if err != nil {
		status := domain.StatusFailed
		errMsg := err.Error()
		_, _ = s.repo.Update(ctx, p.ID, domain.UpdatePaymentInput{
			Status:       &status,
			ErrorMessage: &errMsg,
		})
		return nil, fmt.Errorf("mtn invoice: %w", err)
	}

	return s.repo.Update(ctx, p.ID, domain.UpdatePaymentInput{
		TransactionRef: &refID,
	})
}
