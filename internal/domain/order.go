package domain

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

type OrderItem struct {
	ID        int64
	OrderID   int64
	ProductID int64
	Quantity  int64
	UnitPrice int64 // 注文時点の単価
}

// 明細の小径を返す
func (i *OrderItem) subtotal() int64 {
	return i.UnitPrice * i.Quantity
}

