// Package repository は DB アクセスを担当する。
// domain のエンティティと SQL の相互変換、および
// ドライバ固有のエラーを domain のエラーへ翻訳する責務を持つ。
package repository

import (
	// 【標準ライブラリ】引数の型 context.Context のために使う
	"context"

	// 【標準ライブラリ】戻り値の型 sql.Result / *sql.Rows / *sql.Row と、
	// 下の型チェックで *sql.DB / *sql.Tx を参照するために使う
	"database/sql"
)

// DBTX は *sql.DB と *sql.Tx の両方が持つメソッドのうち、
// repository が実際に呼ぶ3つだけを抜き出した interface。
//
// 【由来】
//
//	interface 名「DBTX」           … 自分で命名（"DB or TX"。sqlc と同じ慣習的な名前）
//	メソッド名・引数・戻り値         … database/sql の *sql.DB / *sql.Tx の既存メソッドを
//	                                 一字一句写したもの。1文字でも変えると満たさなくなる
//
// 【目的】
// repository のフィールドをこの型にしておけば、
// 通常時は *sql.DB を、トランザクション中は *sql.Tx を渡すだけで
// 同じ repository のコードがどちらでも動く。
type DBTX interface {
	// 【database/sql 由来】*sql.DB.ExecContext / *sql.Tx.ExecContext と同じ形
	//   ctx    : キャンセルとタイムアウトを DB まで伝える
	//   query  : 実行する SQL。$1, $2 ... のプレースホルダを含む
	//   args   : プレースホルダに入る値。...any は「任意の型の値を任意個」
	//   戻り値 : sql.Result（RowsAffected で更新行数が取れる）と error
	// 用途 : 行を返さない SQL（INSERT / UPDATE / DELETE）
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)

	// 【database/sql 由来】*sql.DB.QueryContext / *sql.Tx.QueryContext と同じ形
	//   ctx    : 同上
	//   query  : 同上
	//   args   : 同上
	//   戻り値 : *sql.Rows（複数行を1行ずつ読むカーソル）と error
	// 用途 : 複数行を返す SELECT。戻り値の *sql.Rows は必ず Close する
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)

	// 【database/sql 由来】*sql.DB.QueryRowContext / *sql.Tx.QueryRowContext と同じ形
	//   ctx    : 同上
	//   query  : 同上
	//   args   : 同上
	//   戻り値 : *sql.Row のみ。error を返さない
	//            （エラーは *sql.Row の中に保持され、Scan を呼んだときに返る）
	// 用途 : 最大1行を返す SELECT、および RETURNING 付きの INSERT / UPDATE
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// 【Go の慣習】コンパイル時の interface 充足チェック。
//
// *sql.DB と *sql.Tx が DBTX を満たしていることを、コンパイラに確認させる。
// どちらかのメソッドの写し間違いがあれば、この行でコンパイルエラーになる
// （これが無いと、main.go で NewProductRepository(db) を書いた時点まで気づけない）。
//
//	_              : 変数名を捨てる。値を使うためではなく、型チェックのためだけに書く行
//	DBTX           : この型の変数に代入できるか、をコンパイラが調べる
//	(*sql.DB)(nil) : 「*sql.DB 型の nil ポインタ」。中身は要らないので nil でいい
//
// 実行時のコストはゼロ。
var (
	_ DBTX = (*sql.DB)(nil)
	_ DBTX = (*sql.Tx)(nil)
)
