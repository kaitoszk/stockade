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
		name string
		input domain.Product
		setup func(mock sqlmock.Sqlmock)
		wantErr error
	}{
		{
			// TODO
		}
	}

	
}