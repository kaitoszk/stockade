package handler

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/kaitoszk/stockade/internal/domain"
)

func TestStatusFor(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantMsg    string
	}{
		{"ErrInvalidInputは 400", domain.ErrInvalidInput, http.StatusBadRequest, "invalid input"},
		{"ErrUnauthorizedは　401", domain.ErrUnauthorized, http.StatusUnauthorized, "unauthorized"},
		{"ErrForbiddenは 403", domain.ErrForbidden, http.StatusForbidden, "forbidden"},
		{"ErrNotFoundは 404", domain.ErrNotFound, http.StatusNotFound, "not found"},
		{"ErrConflictは 409", domain.ErrConflict, http.StatusConflict, "conflict"},
		{"ErrInsufficientStockは 409", domain.ErrInsufficientStock, http.StatusConflict, "insufficient stock"},
		{"ErrInvalidStatusTransitionは 409", domain.ErrInvalidStatusTransition, http.StatusConflict, "invalid status transition"},
		{
			"wrapされたドメインエラーも判定できる",
			fmt.Errorf("create product: %w", domain.ErrConflict),
			http.StatusConflict,
			"conflict",
		},
		{
			"想定外のエラーは 500",
			errors.New("db down"),
			http.StatusInternalServerError,
			"internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, msg := statusFor(tt.err)
			// t.Logf("【DEBUG】statusFor(%v) → status=%d msg=%q", tt.err, status, msg)

			if status != tt.wantStatus {
				t.Errorf("status = %d, want %d", status, tt.wantStatus)
			}
			if msg != tt.wantMsg {
				t.Errorf("msg = %q, want %q", msg, tt.wantMsg)
			}
		})
	}
	
}
