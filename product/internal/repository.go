package internal

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"thugcorp.io/grocery/product/internal/domain"
)

type ProductRepository interface {
	Create(ctx context.Context, p *domain.Product) (*domain.Product, error)
	GetByID(ctx context.Context, id string) (*domain.Product, error)
	Update(ctx context.Context, p *domain.Product) (*domain.Product, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, businessID, category string, page, pageSize int) ([]*domain.Product, int64, error)
	Search(ctx context.Context, businessID, query string, page, pageSize int) ([]*domain.Product, int64, error)
}

type productRepository struct {
	db *gorm.DB
}

func NewProductRepository(db *gorm.DB) ProductRepository {
	return &productRepository{db: db}
}

func (r *productRepository) Create(ctx context.Context, p *domain.Product) (*domain.Product, error) {
	p.ID = uuid.New().String()
	p.IsActive = true
	if err := r.db.WithContext(ctx).Create(p).Error; err != nil {
		return nil, err
	}
	return p, nil
}

func (r *productRepository) GetByID(ctx context.Context, id string) (*domain.Product, error) {
	var p domain.Product
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&p).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (r *productRepository) Update(ctx context.Context, p *domain.Product) (*domain.Product, error) {
	if err := r.db.WithContext(ctx).Save(p).Error; err != nil {
		return nil, err
	}
	return p, nil
}

func (r *productRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&domain.Product{}).Error
}

func (r *productRepository) List(ctx context.Context, businessID, category string, page, pageSize int) ([]*domain.Product, int64, error) {
	if pageSize <= 0 {
		pageSize = 20
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * pageSize

	q := r.db.WithContext(ctx).Model(&domain.Product{}).Where("business_id = ? AND is_active = true", businessID)
	if category != "" {
		q = q.Where("category = ?", category)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []*domain.Product
	if err := q.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *productRepository) Search(ctx context.Context, businessID, query string, page, pageSize int) ([]*domain.Product, int64, error) {
	if pageSize <= 0 {
		pageSize = 20
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * pageSize
	like := "%" + query + "%"

	q := r.db.WithContext(ctx).Model(&domain.Product{}).
		Where("business_id = ? AND is_active = true AND (name ILIKE ? OR category ILIKE ?)", businessID, like, like)

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []*domain.Product
	if err := q.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}
