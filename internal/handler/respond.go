package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/kaitoszk/stockade/internal/domain"
)

const maxBodyBytes = 1 << 20

type errorResponse struct {
	Error string `json:"error"`
}

// リクエストボディのJSONを引数dstに読み込む
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	// 上限バイト数を超えてreadするとエラーを返す
	// (上限を超えた時に接続を閉じる指示 , ボディ, 上限バイト数)
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	// JSONを読むのに必要
	dec := json.NewDecoder(r.Body)

  // typo時にエラー、priceではくpriseなど
	dec.DisallowUnknownFields()

	// 構文、型の不一致、知らないキー、上限超過でエラーが返る
	return dec.Decode(dst)
}

// ステータスコードとJSONをレスポンスに書く
func writeJSON(w http.ResponseWriter, logger *slog.Logger, status int, v any) {	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	// JSONの書き込みに失敗 = true
	// ステータスコードは送信済みで変えられないので、ログに残すだけにする
	if err := json.NewEncoder(w).Encode(v); err != nil {
		logger.Warn("write response", slog.Any("err", err))
	}
}

// エラーをステータスコードとレスポンスに出す固定の文言に変換する
func statusFor(err error) (int, string) {
	switch {
		case errors.Is(err, domain.ErrInvalidInput):
			return http.StatusBadRequest, "invalid input"
		case errors.Is(err, domain.ErrUnauthorized):
			return http.StatusUnauthorized, "unauthorized"
		case errors.Is(err, domain.ErrForbidden):
			return http.StatusForbidden, "forbidden"
		case errors.Is(err, domain.ErrNotFound):
			return http.StatusNotFound, "not found"
		case errors.Is(err, domain.ErrConflict):
			return http.StatusConflict, "conflict"
		case errors.Is(err, domain.ErrInsufficientStock):
			return http.StatusConflict, "insufficient stock"
		case errors.Is(err, domain.ErrInvalidStatusTransition):
			return http.StatusConflict, "invalid status transition"
		default:
			return http.StatusInternalServerError, "internal server error"
	}
}

// エラーをHTTPレスポンスに変換する
func writeError(w http.ResponseWriter, r *http.Request, logger *slog.Logger, err error) {
	status, msg := statusFor(err)
	fmt.Printf("【DEBUG】writeError: err=%v(type=%T) → status=%d msg=%q\n", err, err, status, msg)

	// 500の時true
	// 原因を調べられるようにログに記録する
	if status == http.StatusInternalServerError {
		logger.Error("unexpected error",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Any("err", err),
		)
	}
	writeJSON(w, logger, status, errorResponse{Error: msg})
}