package repository

import (
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/kaitoszk/stockade/internal/domain"
)

var testTime = time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)

var errDB = errors.New("db down")

func TestProductRepository_Create(t *testing.T) {
	tests := []struct {
		name        string                     // サブテスト名
		setup       func(mock sqlmock.Sqlmock) // このケースで DB がどう応答するかを仕込む
		wantErr     error                      // 期待するエラー。nil なら成功を期待
		wantID      int64                      // 成功時に p.ID に書き戻されるべき値
		wantVersion int64                      // 成功時に p.Version に書き戻されるべき値
	}{
		{
			name: "成功_RETURNINGの値がpに描き戻される",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO products")).
					WithArgs("SKU-001", "テスト商品", 1000).
					WillReturnRows(sqlmock.NewRows([]string{"id", "version", "created_at", "updated_at"}).
						AddRow(1, 1, testTime, testTime))
			},
			wantErr:     nil,
			wantID:      1,
			wantVersion: 1,
		},
		{
			name: "SKU重複：UNIQUE違反をErrConflictに翻訳する",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO products")).
					WithArgs("SKU-001", "テスト商品", 1000).
					WillReturnError(&pgconn.PgError{Code: pgerrcode.UniqueViolation})
			},
			wantErr: domain.ErrConflict,
		},
		{
			name: "DBエラー：ドメインエラーに翻訳せずwrapして返す",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO products")).
					WithArgs("SKU-001", "テスト商品", 1000).
					WillReturnError(errDB)
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

			repo := NewProductRepository(db)
			p := &domain.Product{SKU: "SKU-001", Name: "テスト商品", Price: 1000}
			err = repo.Create(t.Context(), p)
			// TODO:context is obsidian

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}

			if tt.wantErr == nil {
				if p.ID != tt.wantID {
					t.Errorf("p.ID = %d, want %d", p.ID, tt.wantID)
				}
				if p.Version != tt.wantVersion {
					t.Errorf("p.Version = %d, want %d", p.Version, tt.wantVersion)
				}
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

func TestProductRepository_GetByID(t *testing.T) {
	cols := []string{"id", "sku", "name", "price", "version", "created_at", "updated_at"}
	query := regexp.QuoteMeta("SELECT id, sku, name, price, version, created_at, updated_at FROM products")

	tests := []struct {
		name    string
		setup   func(mock sqlmock.Sqlmock)
		want    *domain.Product
		wantErr error
	}{
		{
			name: "成功：1行をProductに詰めて返す",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(query).
					WithArgs(1).
					WillReturnRows(sqlmock.NewRows(cols).
						AddRow(1, "SKU-001", "テスト商品", 1000, 3, testTime, testTime))
			},
			want: &domain.Product{
				ID: 1, SKU: "SKU-001", Name: "テスト商品", Price: 1000, Version: 3, CreatedAt: testTime, UpdatedAt: testTime,
			},
		},
		{
			name: "0行：sql.ErrNoRowsをErrNotFoundに翻訳する",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(query).
					WithArgs(1).
					WillReturnRows(sqlmock.NewRows(cols))
			},
			wantErr: domain.ErrNotFound,
		},
		{
			name: "DBエラー：wrapして返す",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(query).
					WithArgs(1).
					WillReturnError(errDB)
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
			repo := NewProductRepository(db)

			got, err := repo.GetByID(t.Context(), 1)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}

			if tt.wantErr == nil {
				if *got != *tt.want {
					t.Errorf("got %+v, want %+v", *got, *tt.want)
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

func TestProductRepository_Update(t *testing.T) {
	updateQuery := regexp.QuoteMeta("UPDATE products")
	existsQuery := regexp.QuoteMeta("SELECT EXISTS")
	returningCols := []string{"version", "updated_at"}

	tests := []struct {
		name        string
		setup       func(mock sqlmock.Sqlmock)
		wantErr     error
		wantVersion int64
	}{
		{
			name: "成功：versionが一致して更新され、新しいversionが書き戻される",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(updateQuery).
					WithArgs("新しい名前", 1200, 1, 3).
					WillReturnRows(sqlmock.NewRows(returningCols).AddRow(4, testTime))
			},
			wantErr:     nil,
			wantVersion: 4,
		},
		{
			name: "UPDATEがエラー：EXISTSは投げずにwrapして返す",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(updateQuery).
					WithArgs("新しい名前", 1200, 1, 3).
					WillReturnError(errDB)
			},
			wantErr:     errDB,
			wantVersion: 3,
		},
		{
			name: "UPDATEが0行、EXISTSがエラー：wrapして返す",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(updateQuery).
					WithArgs("新しい名前", 1200, 1, 3).
					WillReturnRows(sqlmock.NewRows(returningCols))
				mock.ExpectQuery(existsQuery).
					WithArgs(1).
					WillReturnError(errDB)
			},
			wantErr:     errDB,
			wantVersion: 3,
		},
		{
			name: "UPDATEが0行、商品が存在しない：ErrNotFound",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(updateQuery).
					WithArgs("新しい名前", 1200, 1, 3).
					WillReturnRows(sqlmock.NewRows(returningCols))
				mock.ExpectQuery(existsQuery).
					WithArgs(1).
					WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
			},
			wantErr:     domain.ErrNotFound,
			wantVersion: 3,
		},
		{
			name: "UPDATEが0行、商品は存在する：version不一致なのでErrConflict",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(updateQuery).
					WithArgs("新しい名前", 1200, 1, 3).
					WillReturnRows(sqlmock.NewRows(returningCols))
				mock.ExpectQuery(existsQuery).
					WithArgs(1).
					WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
			},
			wantErr:     domain.ErrConflict,
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
			repo := NewProductRepository(db)

			p := &domain.Product{ID: 1, Name: "新しい名前", Price: 1200, Version: 3}

			err = repo.Update(t.Context(), p)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}

			if p.Version != tt.wantVersion {
				t.Errorf("p.Version = %d, want %d", p.Version, tt.wantVersion)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expextations: %v", err)
			}
		})
	}
}

// TODO:willreturnrows, willreturnerrorの使い分け、227行目エラー、expectationsweremet
