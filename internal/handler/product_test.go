// ProductHandler のテスト。
//
// 本番の「curl → http.Server → router → handler → service → DB」のうち、
// handler 以外をテスト用の部品に置き換えて、handler だけを動かす。
//   curl        → httptest.NewRequest（偽のリクエスト）
//   http.Server → router.ServeHTTP を直接呼ぶ
//   レスポンス   → httptest.NewRecorder（送らずに記録する）
//   service     → fakeProductService（偽の service。DB を使わない）
package handler

import (
	// 【標準ライブラリ】ログの出力先にする（あとで中身を読むため）
	"bytes"
	// 【標準ライブラリ】偽の service のメソッドの引数の型
	"context"
	// 【標準ライブラリ】レスポンスの JSON を Go の構造体に読み戻すため
	"encoding/json"
	// 【標準ライブラリ】「想定外のエラー」を作るため
	"errors"
	// 【標準ライブラリ】wrap されたエラーを作るため（本物の service と同じ形にする）
	"fmt"
	// 【標準ライブラリ】handler に渡すロガー
	"log/slog"
	// 【標準ライブラリ】ステータスコードとメソッド名の定数
	"net/http"
	// 【標準ライブラリ】偽のリクエストと、レスポンスを記録する Recorder
	"net/http/httptest"
	// 【標準ライブラリ】文字列をリクエストのボディ（io.Reader）にするため / ログの検索
	"strings"
	// 【標準ライブラリ】テストの枠組み
	"testing"
	// 【標準ライブラリ】偽の service が書き戻す、固定の時刻
	"time"

	// 【自作】エンティティとドメインエラー
	"github.com/kaitoszk/stockade/internal/domain"
)

// 【自作】偽の service が書き戻す固定の時刻。time.Now() だと毎回変わって比べられないので固定する
var testTime = time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)

// ═════════════════════════════════════════════════════════════════
// 偽の service
// ═════════════════════════════════════════════════════════════════

// fakeProductService は、handler の productService interface を満たす偽物。【自作】
//
// handler が service から受け取るものだけを再現している。
// フィールドは「記録用（何を渡されたか）」と「仕込み用（何を返すか）」の2種類。
type fakeProductService struct {
	got     *domain.Product // 記録用：Create / Update に渡された商品。nil のままなら「呼ばれなかった」
	gotID   int64           // 記録用：GetByID に渡された id。0 のままなら「呼ばれなかった」
	product *domain.Product // 仕込み用：GetByID が返す商品
	err     error           // 仕込み用：返すエラー。nil なら成功させる
}

// Create は偽物の登録。【自作】
// 成功時は、本物なら DB が決める値（ID・Version・時刻）を書き戻す
func (f *fakeProductService) Create(_ context.Context, p *domain.Product) error {
	f.got = p
	if f.err != nil {
		return f.err
	}
	p.ID = 42
	p.Version = 1
	p.CreatedAt = testTime
	p.UpdatedAt = testTime
	return nil
}

// GetByID は偽物の取得。【自作】
// 渡された id を記録して、仕込まれた商品を返す
func (f *fakeProductService) GetByID(_ context.Context, id int64) (*domain.Product, error) {
	f.gotID = id
	if f.err != nil {
		return nil, f.err
	}
	return f.product, nil
}

// Update は偽物の更新。【自作】
// 成功時は、本物なら DB が決める値（RETURNING で返る列）だけを書き戻す。
// ID・Name・Price はクライアントの値なので触らない（handler の詰め替えミスに気づけるように）
func (f *fakeProductService) Update(_ context.Context, p *domain.Product) error {
	f.got = p
	if f.err != nil {
		return f.err
	}
	p.Version++            // DB の version = version + 1 の代わり
	p.SKU = "SKU-001"      // RETURNING で返る列
	p.CreatedAt = testTime // RETURNING で返る列
	p.UpdatedAt = testTime // DB の updated_at = now() の代わり
	return nil
}

// ═════════════════════════════════════════════════════════════════
// POST /products
//
//   出口                        ケース
//   ─────────────────────────  ─────────────────────────────────────
//   JSON を読めない（400）       「JSONが壊れている」「知らないキー」
//   service がエラー            「ErrInvalidInput」「ErrConflict」「想定外」
//   成功（201）                 「成功」
// ═════════════════════════════════════════════════════════════════

func TestProductHandler_Create(t *testing.T) {
	tests := []struct {
		name       string // サブテストの名前
		body       string // 入力：リクエストのボディ
		svcErr     error  // 入力：偽の service に返させるエラー。nil なら成功
		wantStatus int    // 期待：ステータスコード
		wantCalled bool   // 期待：service が呼ばれるか
		wantErrMsg string // 期待：エラーの文言。成功ケースでは空文字
	}{
		{
			name:       "成功：201と、採番されたidを含む商品を返す",
			body:       `{"sku":"SKU-001","name":"テスト商品","price":1000}`,
			wantStatus: http.StatusCreated,
			wantCalled: true,
		},
		{
			name:       "JSONが壊れている：serviceを呼ばずに400",
			body:       `{"sku":`,
			wantStatus: http.StatusBadRequest,
			wantCalled: false,
			wantErrMsg: "invalid JSON",
		},
		{
			// price を prise と typo。DisallowUnknownFields が効いているかを確かめる
			name:       "知らないキーがある：serviceを呼ばずに400",
			body:       `{"sku":"SKU-001","name":"テスト商品","prise":1000}`,
			wantStatus: http.StatusBadRequest,
			wantCalled: false,
			wantErrMsg: "invalid JSON",
		},
		{
			name:       "serviceがErrInvalidInput：400",
			body:       `{"sku":"","name":"テスト商品","price":1000}`,
			svcErr:     domain.ErrInvalidInput,
			wantStatus: http.StatusBadRequest,
			wantCalled: true,
			wantErrMsg: "invalid input",
		},
		{
			name:       "serviceがErrConflict（SKU重複）：409",
			body:       `{"sku":"SKU-001","name":"テスト商品","price":1000}`,
			svcErr:     fmt.Errorf("create product: %w", domain.ErrConflict),
			wantStatus: http.StatusConflict,
			wantCalled: true,
			wantErrMsg: "conflict",
		},
		{
			name:       "serviceが想定外のエラー：500、中身はレスポンスに出さない",
			body:       `{"sku":"SKU-001","name":"テスト商品","price":1000}`,
			svcErr:     errors.New("db down"),
			wantStatus: http.StatusInternalServerError,
			wantCalled: true,
			wantErrMsg: "internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ── 準備 ──
			fake := &fakeProductService{err: tt.svcErr}
			var logBuf bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&logBuf, nil))
			router := NewRouter(NewProductHandler(fake, logger))

			req := httptest.NewRequest(http.MethodPost, "/products", strings.NewReader(tt.body))
			rec := httptest.NewRecorder()

			// ── 実行 ──
			router.ServeHTTP(rec, req)

			// 【DEBUG・確認用】返ってきたものと、service に渡ったものを見る。go test -v のときだけ表示（commit 前に削除）
			t.Logf("【DEBUG】status=%d body=%s got=%+v", rec.Code, rec.Body.String(), fake.got)

			// ── 確認 ──
			// ① ステータス
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			// ② service が呼ばれたか
			if (fake.got != nil) != tt.wantCalled {
				t.Fatalf("service called = %v, want %v", fake.got != nil, tt.wantCalled)
			}
			// ③ エラーのケース：文言。500 ならログに中身が残っているか
			if tt.wantErrMsg != "" {
				var got errorResponse
				if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
					t.Fatalf("decode error response: %v", err)
				}
				if got.Error != tt.wantErrMsg {
					t.Errorf("error = %q, want %q", got.Error, tt.wantErrMsg)
				}
				if tt.wantStatus == http.StatusInternalServerError && !strings.Contains(logBuf.String(), "db down") {
					t.Errorf("log does not contain the error detail: %s", logBuf.String())
				}
				return
			}
			// ④ 成功のケース：書き戻された id、Location、DTO からの詰め替え
			var got productResponse
			if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
				t.Fatalf("decode product response: %v", err)
			}
			if got.ID != 42 || got.Version != 1 {
				t.Errorf("got id=%d version=%d, want id=42 version=1", got.ID, got.Version)
			}
			if loc := rec.Header().Get("Location"); loc != "/products/42" {
				t.Errorf("Location = %q, want %q", loc, "/products/42")
			}
			if fake.got.SKU != "SKU-001" || fake.got.Name != "テスト商品" || fake.got.Price != 1000 {
				t.Errorf("service received %+v", *fake.got)
			}
		})
	}
}

// ═════════════════════════════════════════════════════════════════
// GET /products/{id}
//
//   出口                        ケース
//   ─────────────────────────  ─────────────────────
//   id が数値でない（400）       「idが数値でない」
//   service がエラー            「ErrNotFound」
//   成功（200）                 「成功」
// ═════════════════════════════════════════════════════════════════

func TestProductHandler_GetByID(t *testing.T) {
	// 偽物が返す商品。全ケースで仕込むが、エラーのケースでは偽物が先に err を返すので読まれない
	stub := &domain.Product{ID: 42, SKU: "SKU-001", Name: "テスト商品", Price: 1000, Version: 3,
		CreatedAt: testTime, UpdatedAt: testTime}

	tests := []struct {
		name       string // サブテストの名前
		url        string // 入力：リクエストの URL
		svcErr     error  // 入力：偽の service に返させるエラー。nil なら成功
		wantStatus int    // 期待：ステータスコード
		wantCalled bool   // 期待：service が呼ばれるか
		wantErrMsg string // 期待：エラーの文言。成功ケースでは空文字
	}{
		{
			name:       "成功：200と商品を返す",
			url:        "/products/42",
			wantStatus: http.StatusOK,
			wantCalled: true,
		},
		{
			name:       "idが数値でない：serviceを呼ばずに400",
			url:        "/products/abc",
			wantStatus: http.StatusBadRequest,
			wantCalled: false,
			wantErrMsg: "invalid id",
		},
		{
			name:       "serviceがErrNotFound：404",
			url:        "/products/42",
			svcErr:     fmt.Errorf("get product: %w", domain.ErrNotFound),
			wantStatus: http.StatusNotFound,
			wantCalled: true,
			wantErrMsg: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ── 準備 ──
			// 返す商品は product に仕込む（got は Create / Update 用なので使わない）
			fake := &fakeProductService{product: stub, err: tt.svcErr}
			var logBuf bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&logBuf, nil))
			router := NewRouter(NewProductHandler(fake, logger))

			// GET はボディを送らないので、第3引数は nil
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			rec := httptest.NewRecorder()

			// ── 実行 ──
			router.ServeHTTP(rec, req)

			// 【DEBUG・確認用】返ってきたものと、偽物が記録した id を見る（commit 前に削除）
			t.Logf("【DEBUG】status=%d body=%s gotID=%d", rec.Code, rec.Body.String(), fake.gotID)

			// ── 確認 ──
			// ① ステータス
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			// ② service が呼ばれたか。GetByID は gotID に記録するので、0 かどうかで判定する
			if (fake.gotID != 0) != tt.wantCalled {
				t.Fatalf("service called = %v, want %v", fake.gotID != 0, tt.wantCalled)
			}
			// ③ エラーのケース：文言
			if tt.wantErrMsg != "" {
				var got errorResponse
				if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
					t.Fatalf("decode error response: %v", err)
				}
				if got.Error != tt.wantErrMsg {
					t.Errorf("error = %q, want %q", got.Error, tt.wantErrMsg)
				}
				return
			}
			// ④ 成功のケース：URL の id が届いたか、返した商品がそのまま JSON になったか
			if fake.gotID != 42 {
				t.Errorf("service received id = %d, want 42", fake.gotID)
			}
			var got productResponse
			if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
				t.Fatalf("decode product response: %v", err)
			}
			if got.ID != stub.ID || got.SKU != stub.SKU || got.Name != stub.Name ||
				got.Price != stub.Price || got.Version != stub.Version {
				t.Errorf("response = %+v, want %+v", got, *stub)
			}
			// 時刻は JSON を通ると表現が変わることがあるので、Equal で比べる
			if !got.CreatedAt.Equal(stub.CreatedAt) {
				t.Errorf("created_at = %v, want %v", got.CreatedAt, stub.CreatedAt)
			}
		})
	}
}

// ═════════════════════════════════════════════════════════════════
// PUT /products/{id}
//
//   出口                        ケース
//   ─────────────────────────  ─────────────────────────────────────
//   id が数値でない（400）       「idが数値でない」
//   JSON を読めない（400）       「JSONが壊れている」「skuを送ってきた」
//   service がエラー            「ErrInvalidInput」「ErrNotFound」「ErrConflict」
//   成功（200）                 「成功」
// ═════════════════════════════════════════════════════════════════

func TestProductHandler_Update(t *testing.T) {
	// 「version=3 の時点で読んだ商品を、名前と価格を変えて保存する」ボディ
	validBody := `{"name":"新しい名前","price":1200,"version":3}`

	tests := []struct {
		name       string // サブテストの名前
		url        string // 入力：リクエストの URL
		body       string // 入力：リクエストのボディ
		svcErr     error  // 入力：偽の service に返させるエラー。nil なら成功
		wantStatus int    // 期待：ステータスコード
		wantCalled bool   // 期待：service が呼ばれるか
		wantErrMsg string // 期待：エラーの文言。成功ケースでは空文字
	}{
		{
			name:       "成功：200と、新しいversionを含む商品を返す",
			url:        "/products/42",
			body:       validBody,
			wantStatus: http.StatusOK,
			wantCalled: true,
		},
		{
			name:       "idが数値でない：serviceを呼ばずに400",
			url:        "/products/abc",
			body:       validBody,
			wantStatus: http.StatusBadRequest,
			wantCalled: false,
			wantErrMsg: "invalid id",
		},
		{
			name:       "JSONが壊れている：serviceを呼ばずに400",
			url:        "/products/42",
			body:       `{"name":`,
			wantStatus: http.StatusBadRequest,
			wantCalled: false,
			wantErrMsg: "invalid JSON",
		},
		{
			// JSON としては正しい（最後の } がある）。sku が知らない項目なので 400 になることを確かめる
			name:       "skuを送ってきた：変更できない項目なので、serviceを呼ばずに400",
			url:        "/products/42",
			body:       `{"sku":"NEW-001","name":"新しい名前","price":1200,"version":3}`,
			wantStatus: http.StatusBadRequest,
			wantCalled: false,
			wantErrMsg: "invalid JSON",
		},
		{
			name:       "serviceがErrInvalidInput：400",
			url:        "/products/42",
			body:       `{"name":"","price":1200,"version":3}`,
			svcErr:     domain.ErrInvalidInput,
			wantStatus: http.StatusBadRequest,
			wantCalled: true,
			wantErrMsg: "invalid input",
		},
		{
			name:       "serviceがErrNotFound：404",
			url:        "/products/42",
			body:       validBody,
			svcErr:     fmt.Errorf("edit product: %w", domain.ErrNotFound),
			wantStatus: http.StatusNotFound,
			wantCalled: true,
			wantErrMsg: "not found",
		},
		{
			name:       "serviceがErrConflict（version不一致）：409",
			url:        "/products/42",
			body:       validBody,
			svcErr:     fmt.Errorf("edit product: %w", domain.ErrConflict),
			wantStatus: http.StatusConflict,
			wantCalled: true,
			wantErrMsg: "conflict",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ── 準備 ──
			fake := &fakeProductService{err: tt.svcErr}
			var logBuf bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&logBuf, nil))
			router := NewRouter(NewProductHandler(fake, logger))

			req := httptest.NewRequest(http.MethodPut, tt.url, strings.NewReader(tt.body))
			rec := httptest.NewRecorder()

			// ── 実行 ──
			router.ServeHTTP(rec, req)

			// 【DEBUG・確認用】返ってきたものと、service に渡ったものを見る（commit 前に削除）
			t.Logf("【DEBUG】status=%d body=%s got=%+v", rec.Code, rec.Body.String(), fake.got)

			// ── 確認 ──
			// ① ステータス
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			// ② service が呼ばれたか
			if (fake.got != nil) != tt.wantCalled {
				t.Fatalf("service called = %v, want %v", fake.got != nil, tt.wantCalled)
			}
			// ③ エラーのケース：文言
			if tt.wantErrMsg != "" {
				var got errorResponse
				if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
					t.Fatalf("decode error response: %v", err)
				}
				if got.Error != tt.wantErrMsg {
					t.Errorf("error = %q, want %q", got.Error, tt.wantErrMsg)
				}
				return
			}
			// ④ 成功のケース
			var got productResponse
			if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
				t.Fatalf("decode product response: %v", err)
			}
			// 4 なら「handler が 3 を渡し → 偽物が +1 し → レスポンスまで届いた」
			if got.Version != 4 {
				t.Errorf("response version = %d, want 4", got.Version)
			}
			// 偽物は ID を書き換えないので、handler が URL の id を入れ忘れると 0 になって気づける
			if fake.got.ID != 42 {
				t.Errorf("service received ID = %d, want 42", fake.got.ID)
			}
			// ボディから domain への詰め替え
			if fake.got.Name != "新しい名前" || fake.got.Price != 1200 {
				t.Errorf("service received %+v", *fake.got)
			}
		})
	}
}