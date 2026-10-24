package handler

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/kaitoszk/stockade/internal/domain"
)

var testTime = time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)

// handler product.goのproductService interfaceを満たす偽物
type fakeProductService struct {
	got *domain.Product
	err error
}

// productService interfaceのメソッド
// ダックタイピングできているので、NewProductHandlerに渡せる
func (f *fakeProductService) Create(_ context.Context, p *domain.Product) error {
	f.got = p
	if f.err != nil {
		return f.err
	}

	p.ID = 42
	p.Version = 1
	p.CreatedAt = testTime
	p.UpdatedAt = testTime
	return nil
}

func TestProductHandler_Create(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		svcErr     error
		wantStatus int
		wantCalled bool
		wantErrMsg string
	}{
		{
			name:       "成功：201と、採番されたidを含む商品を返す",
			body:       `{"sku":"SKU-001","name":"テスト商品","price":1000}`,
			wantStatus: http.StatusCreated,
			wantCalled: true,
		},
		{
			name:       "JSONが壊れている：serviceを呼ばずに400",
			body:       `{"sku":`,
			wantStatus: http.StatusBadRequest,
			wantCalled: false,
			wantErrMsg: "invalid JSON",
		},
		{
			name:       "知らないキーがある：serviceを呼ばずに400",
			body:       `{"sku":"SKU-001","name":"テスト商品","prise":1000}`,
			wantStatus: http.StatusBadRequest,
			wantCalled: false,
			wantErrMsg: "invalid JSON",
		},
		{
			name: "serviceがErrInvalidInput：400",
			body: `{"sku":"","name":"テスト商品","price":1000}`,
			svcErr: domain.ErrInvalidInput,
			wantStatus: http.StatusBadRequest,
			wantCalled: true,
			wantErrMsg: "inavalid input",
		},
	}
}
