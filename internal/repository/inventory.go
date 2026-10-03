package repository

import (
	"context"
	"fmt"

	"github.com/kaitoszk/stockade/internal/domain"
)

type InventoryRepository struct {
	db DBTX
}

func NewInventoryRepository(db DBTX) *InventoryRepository {
	return &InventoryRepository{db: db}
}

func (r *InventoryRepository) Create(ctx context.Context, productID int64) error {
	const query = `INSERT INTO inventories (product_id) VALUES ($1)`

	if _, err := r.db.ExecContext(ctx, query, productID); err != nil {
		if isUniqueViolation(err) {
			return domain.ErrConflict
		}
		return fmt.Errorf("insert inventory: %w", err)
	}
	return nil
}

// TODO:上記の理解と、GetByProductIDメソッドから