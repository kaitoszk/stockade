package handler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/kaitoszk/stockade/internal/domain"
)

// handlerがserviceに求める操作
// interfaceは使う側に定義する
// service.ProductServiceはCreateメソッドを持っているので、ダックタイピングが成立する
// 一旦Createだけ
type productService interface {
	Create(ctx context.Context, p *domain.Product) error
}

// [/products]系のエンドポイントを担当する
type ProductHandler struct {
	svc    productService
	logger *slog.Logger
}

// コンストラクタ
func NewProductHandler(svc productService, logger *slog.Logger) *ProductHandler {
	return &ProductHandler{svc: svc, logger: logger}
}

// クライアントに書かせて良い項目だけ（id, versionはない）
type createProductRequest struct {
	SKU   string `json:"sku"`
	Name  string `json:"name"`
	Price int64  `json:"price"`
}

// 商品を返す時のJSONの形
type productResponse struct {
	ID        int64     `json:"id"`
	SKU       string    `json:"sku"`
	Name      string    `json:"name"`
	Price     int64     `json:"price"`
	Version   int64     `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// domain.ProductをレスポンスのDTOに詰め替える
func newProductResponse(p *domain.Product) productResponse {
	return productResponse{
		ID:        p.ID,
		SKU:       p.SKU,
		Name:      p.Name,
		Price:     p.Price,
		Version:   p.Version,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}

func (h *ProductHandler) Create(w http.ResponseWriter, r *http.Request) {
	// クライアントが書いていいもの
	var req createProductRequest

	// JSONとして読めるか
	// trueになる：JSONとして壊れている、型が違う、キー名がおかしい、容量オーバー
	if err := decodeJSON(w, r, &req); err != nil {
		fmt.Printf("【DEBUG】handler.Create: decode error=%v\n", err)

		writeJSON(w, h.logger, http.StatusBadRequest, errorResponse{Error: "invalid JSON"})
		return
	}

	fmt.Printf("【DEBUG】handler.Create: req=%+v\n", req)

	// DTOからdomainに詰め替え
	p := &domain.Product{SKU: req.SKU, Name: req.Name, Price: req.Price}

	// serviceが成功したか
	// trueになる：入力が不正、SKUが重複、接続断など
	if err := h.svc.Create(r.Context(), p); err != nil {
		writeError(w, r, h.logger, err)
		return
	}
	fmt.Printf("【DEBUG】handler.Create: created p=%+v\n", p)

	// 成功：201Createdの時は、作ったリソースのURLをLocationでヘッダーに返すのがHTTPの慣習
	w.Header().Set("Location", fmt.Sprintf("/products/%d", p.ID))

	// serviceがpに書き戻したものを返す
	writeJSON(w, h.logger, http.StatusCreated, newProductResponse(p))
}

