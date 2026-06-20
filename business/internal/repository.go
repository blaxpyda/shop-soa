package internal

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"thugcorp.io/grocery/business/internal/domain"
)

type BusinessRepository interface {
	Create(ctx context.Context, input domain.CreateBusinessInput) (*domain.Business, error)
	GetByID(ctx context.Context, id string) (*domain.Business, error)
	Update(ctx context.Context, id string, input domain.UpdateBusinessInput) (*domain.Business, error)
	Delete(ctx context.Context, id string) error
}

type businessRepository struct {
	db *gorm.DB
}

func NewBusinessRepository(db *gorm.DB) BusinessRepository {
	return &businessRepository{db: db}
}

func (r *businessRepository) Create(ctx context.Context, input domain.CreateBusinessInput) (*domain.Business, error) {
	b := &domain.Business{
		ID:          uuid.New().String(),
		OwnerID:     input.OwnerID,
		Name:        input.Name,
		Description: input.Description,
		Email:       input.Email,
		Phone:       input.Phone,
		Address:     input.Address,
		City:        input.City,
		Country:     input.Country,
		IsActive:    true,
	}
	if err := r.db.WithContext(ctx).Create(b).Error; err != nil {
		return nil, err
	}
	return b, nil
}

func (r *businessRepository) GetByID(ctx context.Context, id string) (*domain.Business, error) {
	var b domain.Business
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&b).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &b, nil
}

func (r *businessRepository) Update(ctx context.Context, id string, input domain.UpdateBusinessInput) (*domain.Business, error) {
	updates := map[string]any{}
	if input.Name != "" {
		updates["name"] = input.Name
	}
	if input.Description != "" {
		updates["description"] = input.Description
	}
	if input.Email != "" {
		updates["email"] = input.Email
	}
	if input.Phone != "" {
		updates["phone"] = input.Phone
	}
	if input.Address != "" {
		updates["address"] = input.Address
	}
	if input.City != "" {
		updates["city"] = input.City
	}
	if input.Country != "" {
		updates["country"] = input.Country
	}
	if input.IsActive != nil {
		updates["is_active"] = *input.IsActive
	}

	result := r.db.WithContext(ctx).Model(&domain.Business{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, errors.New("business not found")
	}
	return r.GetByID(ctx, id)
}

func (r *businessRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&domain.Business{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("business not found")
	}
	return nil
}
