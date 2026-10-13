package service

import (
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/kaitoszk/stockade/internal/domain"
)

func TestInventoryService_GetByProductID(t *testing.T) {
	query := regexp.QuoteMeta("SELECT product_id, quantity_on_hand, quantity_reserved, updated_at FROM inventories")
	cols := []string{"product_id", "quantity_on_hand", "quantity_reserved", "updated_at"}

	tests := []struct {
		name string
		setup func(mock sqlmock.Sqlmock)
		wantInventory *domain.Inventory
		wantErr error
	}{
		{
			name: "成功：repositoryの結果をそのまま返す",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(query).
					WithArgs(1).
					WillReturnRows(sqlmock.NewRows(cols).AddRow(1, 10, 3, testTime))
			},
			wantInventory: &domain.Inventory{
				ProductID: 1,
				QuantityOnHand: 10,
				QuantityReserved: 3,
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
			svc := NewInventoryService(db)
			got, err := svc.GetByProductID(t.Context(), 1)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}

			if tt.wantErr == nil && *got != *tt.wantInventory {
				t.Errorf("got %+v, want %+v", *got, *tt.wantInventory)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}