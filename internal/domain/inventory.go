package domain

import "time"

// 在庫、Productと1:1
type Inventory struct {
	ProductID int64
	// 物理在庫
	QuantityOnHand int64
	// 注文されたがまだ出荷していない数
	QuantityReserved int64
	UpdatedAt time.Time
}

// 販売可能数を返す
func (i *Inventory) Available() int64 {
	return i.QuantityOnHand - i.QuantityReserved
}