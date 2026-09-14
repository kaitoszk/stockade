package domain

import "errors"

// ドメインエラー
// ドメインエラーとは、業務上起こりうる失敗を指す
// DB接続失敗のような技術的エラーはここに含めない
var (
	// 404：存在なし
	ErrNotFound = errors.New("not found")

	// 403：権限なし
	ErrForbidden = errors.New("forbidden")

	// 401：認証情報なし（不正）
	ErrUnauthorized = errors.New("unauthorized")

	// 409：在庫数より要求数より小さい
	ErrInsufficientStock = errors.New("insufficient stock")

	// 409：状態が変化してしまい処理できない
	ErrConflict = errors.New("conflict")

	// 400：入力値がビジネスルールを満たさない（数量0など）
	ErrInvalidInput = errors.New("invalid input")

	// 409：許されていないステータス遷移（注文キャンセルなど）
	ErrInvalidStatusTransition = errors.New("invalid status transition")
)
