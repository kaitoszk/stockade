package service

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/kaitoszk/stockade/internal/domain"
	"github.com/kaitoszk/stockade/internal/repository"
)

type InventoryService struct {
	db *sql.DB
}

func NewInventoryService(db *sql.DB) *InventoryService {
	return &InventoryService{db: db}
} 

func (s *InventoryService) GetByProductID(ctx context.Context, productID int64) (*domain.Inventory, error) {
	repo := repository.NewInventoryRepository(s.db)
	inv, err := repo.GetByProductID(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("get inventory: %w", err)
	}
	return inv, nil
}