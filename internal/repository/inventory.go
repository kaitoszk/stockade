package repository

import (
	"context"
	"database/sql"
	"errors"
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

func (r *InventoryRepository) GetByProductID(ctx context.Context, productID int64) (*domain.Inventory, error) {
	const query = `
		SELECT product_id, quantity_on_hand, quantity_reserved, updated_at
		FROM inventories
		WHERE product_id = $1
	`

	var inv domain.Inventory
	err := r.db.QueryRowContext(ctx, query, productID).Scan(
		&inv.ProductID,
		&inv.QuantityOnHand,
		&inv.QuantityReserved,
		&inv.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("select inventory: %w", err)
	}
	return &inv, err
}

func (r *InventoryRepository) AddStock(ctx context.Context, productID, qty int64) (*domain.Inventory, error) {
	const query = `
		UPDATE inventories
		SET quantity_on_hand = quantity_on_hand + $1,
		    updated_at = now()
		WHERE product_id = $2
		RETURNING product_id, quantity_on_hand, quantity_reserved, updated_at
	`
	var inv domain.Inventory
	err := r.db.QueryRowContext(ctx, query, qty, productID).Scan(
		&inv.ProductID,
		&inv.QuantityOnHand,
		&inv.QuantityReserved,
		&inv.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("add stock: %w", err)
	}
	return &inv, nil
}
