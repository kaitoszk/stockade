package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
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
			name:       "serviceがErrInvalidInput：400",
			body:       `{"sku":"","name":"テスト商品","price":1000}`,
			svcErr:     domain.ErrInvalidInput,
			wantStatus: http.StatusBadRequest,
			wantCalled: true,
			wantErrMsg: "invalid input",
		},
		{
			name:       "serviceがErrConflict（SKU重複）：409",
			body:       `{"sku":"SKU-001","name":"テスト商品","price":1000}`,
			svcErr:     fmt.Errorf("create product: %w", domain.ErrConflict),
			wantStatus: http.StatusConflict,
			wantCalled: true,
			wantErrMsg: "conflict",
		},
		{
			name:       "serviceが想定外のエラー：500、中身はレスポンスに出さない",
			body:       `{"sku":"SKU-001","name":"テスト商品","price":1000}`,
			svcErr:     errors.New("db down"),
			wantStatus: http.StatusInternalServerError,
			wantCalled: true,
			wantErrMsg: "internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeProductService{err: tt.svcErr}

			var logBuf bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&logBuf, nil))

			h := NewProductHandler(fake, logger)
			router := NewRouter(h)
			// TODO：上記の理解をしてから、以下に進む
			// slog.New、slog.NewTextHandler、bytes.Bufferをobsidianでまとめる

			req := httptest.NewRequest(http.MethodPost, "/products", strings.NewReader(tt.body))
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			t.Logf("【DEBUG】status=%d body=%s", rec.Code, rec.Body.String())
			t.Logf("【DEBUG】location=%q", rec.Header().Get("Location"))
			t.Logf("【DEBUG】log=%s", logBuf.String())
			t.Logf("【DEBUG】service got=%+v", fake.got)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}

			if (fake.got != nil) != tt.wantCalled {
				t.Fatalf("service called = %v, want %v", fake.got != nil, tt.wantCalled)
			}

			if tt.wantErrMsg != "" {
				var got errorResponse
				if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
					t.Fatalf("decode error response: %v", err)
				}

				if got.Error != tt.wantErrMsg {
					t.Errorf("error = %q, want %q", got.Error, tt.wantErrMsg)
				}

				if tt.wantStatus == http.StatusInternalServerError {
					if !strings.Contains(logBuf.String(), "db down") {
						t.Errorf("log does not contain the error detail: %s", logBuf.String())
					}
				}
				return
			}

			var got productResponse
			if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
				t.Fatalf("decode product response: %v", err)
			}

			if got.ID != 42 || got.Version != 1 {
				t.Errorf("got id=%d version=%d, want id=42 version=1", got.ID, got.Version)
			}

			if loc := rec.Header().Get("Location"); loc != "/products/42" {
				t.Errorf("Location = %q, want %q", loc, "/products/42")
			}

			if fake.got.SKU != "SKU-001" || fake.got.Name != "テスト商品" || fake.got.Price != 1000 {
				t.Errorf("service received %+v", *fake.got)
			}
			
		})
	}
}
