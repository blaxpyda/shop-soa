package internal

import (
	"context"
	"errors"

	"thugcorp.io/grocery/business/internal/domain"
)

type BusinessService interface {
	CreateBusiness(ctx context.Context, ownerID string, input domain.CreateBusinessInput) (*domain.Business, error)
	GetBusiness(ctx context.Context, id string) (*domain.Business, error)
	UpdateBusiness(ctx context.Context, callerID, callerRole, id string, input domain.UpdateBusinessInput) (*domain.Business, error)
	DeleteBusiness(ctx context.Context, callerID, callerRole, id string) error
}

type businessService struct {
	repo BusinessRepository
}

func NewBusinessService(repo BusinessRepository) BusinessService {
	return &businessService{repo: repo}
}

func (s *businessService) CreateBusiness(ctx context.Context, ownerID string, input domain.CreateBusinessInput) (*domain.Business, error) {
	if input.Name == "" {
		return nil, errors.New("business name is required")
	}
	input.OwnerID = ownerID
	return s.repo.Create(ctx, input)
}

func (s *businessService) GetBusiness(ctx context.Context, id string) (*domain.Business, error) {
	b, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, errors.New("business not found")
	}
	return b, nil
}

func (s *businessService) UpdateBusiness(ctx context.Context, callerID, callerRole, id string, input domain.UpdateBusinessInput) (*domain.Business, error) {
	b, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, errors.New("business not found")
	}

	if !isOwner(callerID, callerRole, b.OwnerID) {
		return nil, errors.New("forbidden: only the owner can update this business")
	}

	return s.repo.Update(ctx, id, input)
}

func (s *businessService) DeleteBusiness(ctx context.Context, callerID, callerRole, id string) error {
	b, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if b == nil {
		return errors.New("business not found")
	}

	if !isOwner(callerID, callerRole, b.OwnerID) {
		return errors.New("forbidden: only the owner can delete this business")
	}

	return s.repo.Delete(ctx, id)
}

func isOwner(callerID, callerRole, ownerID string) bool {
	return callerID == ownerID || callerRole == "super-admin"
}
