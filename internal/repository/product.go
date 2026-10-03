package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/kaitoszk/stockade/internal/domain"
)

type ProductRepository struct {
	db DBTX
}

func NewProductRepository(db DBTX) *ProductRepository {
	return &ProductRepository{db: db}
}

func (r *ProductRepository) Create(ctx context.Context, p *domain.Product) error {
	const query = `
	  INSERT INTO products (sku, name, price)
	  VALUES ($1, $2, $3) RETURNING id, version, created_at, updated_at
	`
	err := r.db.QueryRowContext(ctx, query, p.SKU, p.Name, p.Price).
		Scan(&p.ID, &p.Version, &p.CreatedAt, &p.UpdatedAt)

	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrConflict
		}
		return fmt.Errorf("insert product: %w", err)
	}
	return nil
}

func (r *ProductRepository) GetByID(ctx context.Context, id int64) (*domain.Product, error) {
	const query = `SELECT id, sku, price, version, created_at, updated_at FROM products WHERE id = $1`

	var p domain.Product
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&p.ID,
		&p.SKU,
		&p.Name,
		&p.Price,
		&p.Version,
		&p.CreatedAt,
		&p.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("select product: %w", err)
	}
	return &p, nil
}

func (r *ProductRepository) Update(ctx context.Context, p *domain.Product) error {
	const query = `
		UPDATE products SET name = $1, price = $2, version = version + 1, updated_at = now()
		WHERE id = $3 AND version = $4 RETURNING version, updated_at
	`

	err := r.db.QueryRowContext(ctx, query, p.Name, p.Price, p.ID, p.Version).
		Scan(&p.Version, &p.UpdatedAt)

	if err == nil {
		return nil
	}

	if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("udpate product: %w", err)
	}

	const existsQuery = `SELECT EXISTS(SELECT 1 FROM products WHERE id = $1)`
	var exists bool
	if err := r.db.QueryRowContext(ctx, existsQuery, p.ID).Scan(&exists); err != nil {
		return fmt.Errorf("check product existence: %w", err)
	}
	if !exists {
		return domain.ErrNotFound
	}
	return domain.ErrConflict
}
