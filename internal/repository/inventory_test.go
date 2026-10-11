package repository

import (
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/kaitoszk/stockade/internal/domain"
)

func TestInventoryRepository_Create(t *testing.T) {
	query := regexp.QuoteMeta("INSERT INTO inventories")

	tests := []struct {
		name    string
		setup   func(mock sqlmock.Sqlmock)
		wantErr error
	}{
		{
			name: "成功：在庫行が作られる",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(query).
					WithArgs(1).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr: nil,
		},
		{
			name: "重複：PRIMARY KEY重複をErrConflictに翻訳する",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(query).
					WithArgs(1).
					WillReturnError(&pgconn.PgError{Code: pgerrcode.UniqueViolation})
			},
			wantErr: domain.ErrConflict,
		},
		{
			name: "DBエラー：ドメインエラーに翻訳せずwrapして返す",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(query).
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
			repo := NewInventoryRepository(db)
			err = repo.Create(t.Context(), 1)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

func TestInventoryReposity_GetByProductID(t *testing.T) {
	cols := []string{"product_id", "quantity_on_hand", "quantity_reserved", "updated_at"}
	query := regexp.QuoteMeta("SELECT product_id, quantity_on_hand, quantity_reserved, updated_at FROM inventories")

	tests := []struct {
		name      string
		setup     func(mock sqlmock.Sqlmock)
		wantStock *domain.Inventory
		wantErr   error
	}{
		{
			name: "成功：1行をInventoryに詰めて返す",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(query).
					WithArgs(1).
					WillReturnRows(sqlmock.NewRows(cols).
						AddRow(1, 10, 3, testTime))
			},
			wantStock: &domain.Inventory{
				ProductID: 1,
				QuantityOnHand: 10,
				QuantityReserved: 3,
				UpdatedAt: testTime,
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
			repo := NewInventoryRepository(db)
			got, err := repo.GetByProductID(t.Context(), 1)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}

			if tt.wantErr == nil {
				if *got != *tt.wantStock {
					t.Errorf("got %+v, want%+v", *got, *tt.wantStock)
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}
