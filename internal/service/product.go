package service

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/kaitoszk/stockade/internal/domain"
	"github.com/kaitoszk/stockade/internal/repository"
)

type ProductService struct {
	db *sql.DB
}

func NewProductService(db *sql.DB) *ProductService {
	return &ProductService{db: db}
}

func (s *ProductService) Create(ctx context.Context, p *domain.Product) error {
	if p.SKU == "" || p.Name == "" || p.Price < 0 {
		return domain.ErrInvalidInput
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	productRepo := repository.NewProductRepository(tx)
	inventoryRepo := repository.NewInventoryRepository(tx)

	if err := productRepo.Create(ctx, p); err != nil {
		return fmt.Errorf("create product: %w", err)
	}
	if err := inventoryRepo.Create(ctx, p.ID); err != nil {
		return fmt.Errorf("create inventory: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

func (s *ProductService) GetByID(ctx context.Context, id int64) (*domain.Product, error) {
	repo := repository.NewProductRepository(s.db)
	p, err := repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get product: %w", err)
	}
	return p, nil
}

func (s*ProductService) Update(ctx context.Context, p *domain.Product) error {
	if p.Name == "" || p.Price < 0 || p.Version < 1 {
		return domain.ErrInvalidInput
	}
	repo := repository.NewProductRepository(s.db)
	if err := repo.Update(ctx, p); err != nil {
		return fmt.Errorf("update product: %w", err)
	}
	return nil
}