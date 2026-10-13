package service

import (
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/kaitoszk/stockade/internal/domain"
)

var testTime = time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
var errDB = errors.New("db down")

func TestProductService_Create(t *testing.T) {
	productQuery := regexp.QuoteMeta("INSERT INTO products")
	inventoryQuery := regexp.QuoteMeta("INSERT INTO inventories")

	returningCols := []string{"id", "version", "created_at", "updated_at"}
	valid := domain.Product{SKU: "SKU-001", Name: "テスト商品", Price: 1000}

	tests := []struct {
		name    string
		input   domain.Product
		setup   func(mock sqlmock.Sqlmock)
		wantErr error
	}{
		{
			name:  "成功：BEGIN->商品->在庫->COMMITの順に実行される",
			input: valid,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(productQuery).
					WithArgs("SKU-001", "テスト商品", 1000).
					WillReturnRows(sqlmock.NewRows(returningCols).AddRow(42, 1, testTime, testTime))
				mock.ExpectExec(inventoryQuery).
					WithArgs(42).
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
			},
			wantErr: nil,
		},
		{
			name:    "SKUが空：BEGINせずにErrInvalidInput",
			input:   domain.Product{SKU: "", Name: "テスト商品", Price: 1000},
			setup:   func(mock sqlmock.Sqlmock) {},
			wantErr: domain.ErrInvalidInput,
		},
		{
			name:    "商品名が空：BEGINせずにErrInvalidInput",
			input:   domain.Product{SKU: "SKU-001", Name: "", Price: 1000},
			setup:   func(mock sqlmock.Sqlmock) {},
			wantErr: domain.ErrInvalidInput,
		},
		{
			name:    "価格が負の値：BEGINせずにErrInvalidInput",
			input:   domain.Product{SKU: "SKU-001", Name: "テスト商品", Price: -1},
			setup:   func(mock sqlmock.Sqlmock) {},
			wantErr: domain.ErrInvalidInput,
		},
		{
			name:  "BEGINが失敗：何も実行せずにwrapして返す",
			input: valid,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin().WillReturnError(errDB)
			},
			wantErr: errDB,
		},
		{
			name:  "商品のINSERTが失敗：ROLLBACKされる",
			input: valid,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(productQuery).
					WithArgs("SKU-001", "テスト商品", 1000).
					WillReturnError(errDB)
				mock.ExpectRollback()
			},
			wantErr: errDB,
		},
		{
			name:  "在庫のINSERTが失敗：商品のINSERTごとROLLBACKされる",
			input: valid,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(productQuery).
					WithArgs("SKU-001", "テスト商品", 1000).
					WillReturnRows(sqlmock.NewRows(returningCols).AddRow(42, 1, testTime, testTime))
				mock.ExpectExec(inventoryQuery).
					WithArgs(42).
					WillReturnError(errDB)
				mock.ExpectRollback()
			},
			wantErr: errDB,
		},
		{
			name:  "COMMITが失敗：wrapして返す",
			input: valid,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(productQuery).
					WithArgs("SKU-001", "テスト商品", 1000).
					WillReturnRows(sqlmock.NewRows(returningCols).AddRow(42, 1, testTime, testTime))
				mock.ExpectExec(inventoryQuery).
					WithArgs(42).
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit().WillReturnError(errDB)
			},
			wantErr: errDB,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("sqlmock.New: %v", err)
			}
			defer db.Close()
			tt.setup(mock)

			svc := NewProductService(db)
			p := tt.input
			err = svc.Create(t.Context(), &p)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}

			if tt.wantErr == nil && p.ID != 42 {
				t.Errorf("p.ID = %d, want 42", p.ID)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

func TestProductService_GetByID(t *testing.T) {
	query := regexp.QuoteMeta("SELECT id, sku, name, price, version, created_at, updated_at FROM products")
	cols := []string{"id", "sku", "name", "price", "version", "created_at", "upated_at"}

	tests := []struct {
		name        string
		setup       func(mock sqlmock.Sqlmock)
		wantProduct *domain.Product
		wantErr     error
	}{
		{
			name: "成功：repositoryの結果をそのまま返す",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(query).
					WithArgs(1).
					WillReturnRows(sqlmock.NewRows(cols).
						AddRow(1, "SKU-001", "テスト商品", 1000, 3, testTime, testTime))
			},
			wantProduct: &domain.Product{
				ID:        1,
				SKU:       "SKU-001",
				Name:      "テスト商品",
				Price:     1000,
				Version:   3,
				CreatedAt: testTime,
				UpdatedAt: testTime,
			},
		},
		{
			name: "存在しない：ErrNotFoundがwrapされて届く",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(query).
					WithArgs(1).
					WillReturnRows(sqlmock.NewRows(cols))
			},
			wantErr: domain.ErrNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("sqlmock.New: %v", err)
			}
			defer db.Close()
			tt.setup(mock)
			svc := NewProductService(db)
			got, err := svc.GetByID(t.Context(), 1)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}

			if tt.wantErr == nil && *got != *tt.wantProduct {
				t.Errorf("got %+v, want %+v", *got, *tt.wantProduct)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

func TestProductService_Update(t *testing.T) {
	updateQuery := regexp.QuoteMeta("UPDATE products")
	existsQuery := regexp.QuoteMeta("SELECT EXISTS")
	returningCols := []string{"version", "updated_at"}
	valid := domain.Product{
		ID:      1,
		Name:    "新しい名前",
		Price:   1200,
		Version: 3,
	}

	tests := []struct {
		name        string
		input       domain.Product
		setup       func(mock sqlmock.Sqlmock)
		wantErr     error
		wantVersion int64
	}{
		{
			name: "成功：新しいversionが描き戻される",
			input: valid,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(updateQuery).
					WithArgs("新しい名前", 1200, 1, 3).
					WillReturnRows(sqlmock.NewRows(returningCols).AddRow(4, testTime))
			},
			wantErr: nil,
			wantVersion: 4,
		},
		{
			name: "商品が空：SQLを投げずにErrInvalidInput",
			input: domain.Product{ID: 1, Name: "", Price: 1200, Version: 3},
			setup: func(mock sqlmock.Sqlmock) {},
			wantErr: domain.ErrInvalidInput,
			wantVersion: 3,
		},
		{
			name: "価格が負：SQLを投げずにErrInvalidInput",
			input: domain.Product{ID: 1, Name: "新しい名前", Price: -1, Version: 3},
			setup: func(mock sqlmock.Sqlmock) {},
			wantErr: domain.ErrInvalidInput,
			wantVersion: 3,
		},
		{
			name: "Versionが0（送り忘れ）：409ではなくErrInvalidInput",
			input: domain.Product{ID: 1, Name: "新しい名前", Price: 1200},
			setup: func(mock sqlmock.Sqlmock) {},
			wantErr: domain.ErrInvalidInput,
			wantVersion: 0,
		},
		{
			name: "version不一致：ErrConflictがwrapされて届く",
			input: valid,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(updateQuery).
					WithArgs("新しい名前", 1200, 1, 3).
					WillReturnRows(sqlmock.NewRows(returningCols))
				mock.ExpectQuery(existsQuery).
					WithArgs(1).
					WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
			},
			wantErr: domain.ErrConflict,
			wantVersion: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("sqlmock.New: %v", err)
			}
			defer db.Close()

			tt.setup(mock)
			svc := NewProductService(db)
			p := tt.input
			err = svc.Update(t.Context(), &p)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}

			if p.Version != tt.wantVersion {
				t.Errorf("p.Version = %d, want %d", p.Version, tt.wantVersion)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectation: %v", err)
			}
		})
	}
}
