package internal

import (
	"context"
	"errors"

	"thugcorp.io/grocery/product/internal/domain"
)

type ProductService interface {
	CreateProduct(ctx context.Context, input CreateProductInput) (*domain.Product, error)
	GetProduct(ctx context.Context, id string) (*domain.Product, error)
	UpdateProduct(ctx context.Context, input UpdateProductInput) (*domain.Product, error)
	DeleteProduct(ctx context.Context, id string) error
	ListProducts(ctx context.Context, businessID, category string, page, pageSize int) ([]*domain.Product, int64, error)
	SearchProducts(ctx context.Context, businessID, query string, page, pageSize int) ([]*domain.Product, int64, error)
}

type CreateProductInput struct {
	BusinessID      string
	Name            string
	Category        string
	Price           float64
	Quantity        float64
	ImageURL        string
}

type UpdateProductInput struct {
	ID       string
	Name     string
	Category string
	Price    float64
	Quantity        float64
	ImageURL string
}

type productService struct {
	repo ProductRepository
}

func NewProductService(repo ProductRepository) ProductService {
	return &productService{repo: repo}
}

func (s *productService) CreateProduct(ctx context.Context, input CreateProductInput) (*domain.Product, error) {
	p := &domain.Product{
		BusinessID:      input.BusinessID,
		Name:            input.Name,
		Category:        input.Category,
		Price:           input.Price,
		Quantity:        input.Quantity,
		ImageURL:        input.ImageURL,
	}
	return s.repo.Create(ctx, p)
}

func (s *productService) GetProduct(ctx context.Context, id string) (*domain.Product, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, errors.New("product not found")
	}
	return p, nil
}

func (s *productService) UpdateProduct(ctx context.Context, input UpdateProductInput) (*domain.Product, error) {
	p, err := s.repo.GetByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, errors.New("product not found")
	}

	if input.Name != "" {
		p.Name = input.Name
	}
	if input.Category != "" {
		p.Category = input.Category
	}
	if input.Price != 0 {
		p.Price = input.Price
	}
	if input.Quantity != 0 {
		p.Quantity = input.Quantity
	}
	if input.ImageURL != "" {
		p.ImageURL = input.ImageURL
	}

	return s.repo.Update(ctx, p)
}

func (s *productService) DeleteProduct(ctx context.Context, id string) error {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if p == nil {
		return errors.New("product not found")
	}
	return s.repo.Delete(ctx, id)
}

func (s *productService) ListProducts(ctx context.Context, businessID, category string, page, pageSize int) ([]*domain.Product, int64, error) {
	return s.repo.List(ctx, businessID, category, page, pageSize)
}

func (s *productService) SearchProducts(ctx context.Context, businessID, query string, page, pageSize int) ([]*domain.Product, int64, error) {
	return s.repo.Search(ctx, businessID, query, page, pageSize)
}
