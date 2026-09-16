package domain

import "time"

// 注文ステータスを表す
type OrderStatus string

const (
	// 注文直後の状態
	OrderStatusPending OrderStatus = "pending"

	// 出荷確定の状態
	OrderStatusShipped OrderStatus = "shipped"

	// キャンセルの状態
	OrderStatusCancelled OrderStatus = "cancelled"
)

// OrderStatusが定義済みの値かを判定する
func (s OrderStatus) Isvalid() bool {
	switch s {
	case OrderStatusPending, OrderStatusShipped, OrderStatusCancelled:
		return true
	default:
		return false
	}
}

// 受注明細
type OrderItem struct {
	ID        int64
	OrderID   int64
	ProductID int64
	Quantity  int64
	UnitPrice int64 // 注文時点の単価
}

// 明細の小計を返す
func (i *OrderItem) subtotal() int64 {
	return i.UnitPrice * i.Quantity
}

// 受注
type Order struct {
	ID        int64
	userID    int64
	Status    OrderStatus
	Items     []OrderItem
	CreatedAt time.Time
	UpdatedAt time.Time
}

// 注文全体の合計金額を返す
func (o *Order) TotalPrice() int64 {
	var total int64
	for i := range o.Items {
		total += o.Items[i].subtotal()
	}
	return total
}

func (o *Order) CanTransitionTo(to OrderStatus) bool {
	switch o.Status {
	case OrderStatusPending:
		return to == OrderStatusShipped || to == OrderStatusCancelled
	case OrderStatusShipped, OrderStatusCancelled:
		return false
	default:
		return false
	}
}

// 自分の注文か
func (o *Order) IsOwnedBy(userID int64) bool {
	return o.userID == userID
}