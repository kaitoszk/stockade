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
					WillReturnRows(sqlmock.NewRows([]string{"id", "version", "created_at", "udpated_at"}).
						AddRow(1, 1, testTime, testTime))
			},
			wantErr: nil,
			wantID: 1,
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
				if p.Version != int(tt.wantVersion) {
					t.Errorf("p.Version = %d, want %d", p.Version, tt.wantVersion)
				}
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

// TODO:アプリの全体像把握、上記テストコードの理解、続き