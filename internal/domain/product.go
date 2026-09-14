package domain

import "time"

// 商品マスタ
type Product struct {
	ID        int64
	SKU       string
	Name      string
	Price     int64
	Version   int
	CreatedAt time.Time
	UpdatedAt time.Time
}
