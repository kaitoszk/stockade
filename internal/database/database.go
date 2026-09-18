// database packageはPostgreSQLへの接続を確立する責務だけ
// クエリの実行はrepository packageの仕事
package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// 接続プール
type Config struct {
	// 接続情報
	DSN string

	// 同時に開く接続数の上限
	MaxOpenConns int

	// 使われずにプールへ保持される接続数の上限
	MaxIdleConns int

	// 一本の接続を使い続ける最大時間
	ConnMaxLifetime time.Duration

	// アイドル状態の接続を保持する最大時間
	ConnMaxIdleTime time.Duration
}

// DSN以外を埋めた構造体Configを返す
// 値の根拠は「ローカル開発 + 並行テストが動く程度」。
// 本番では負荷試験の結果に合わせて調整する種類の値なので、ここでの数字はあくまで初期値。
func DefaultConfig(dsn string) Config {
	return Config{
		DSN: dsn,
		MaxOpenConns: 25,
		MaxIdleConns: 25,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 1 * time.Minute,
	}
}

func Open(ctx context.Context, cfg Config) (*sql.DB, error) {
	// sql.Openは接続はしない
	db, err := sql.Open("pgx", cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	
}