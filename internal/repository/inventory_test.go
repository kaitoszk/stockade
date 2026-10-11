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
		name string
		setup func(mock sqlmock.Sqlmock)
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