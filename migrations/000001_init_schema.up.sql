-- ============================================================
-- users : 認証主体。ロールベース認可の判定元
-- ============================================================
CREATE TABLE users (
    -- BIGSERIAL は内部的に BIGINT + sequence。将来の行数上限を気にしなくて済む
    id            BIGSERIAL    PRIMARY KEY,
    -- UNIQUE を張ることで「同じメールで2回登録」を DB 側で弾く。
    -- アプリ側の存在チェックは競合するので、最終的な保証はここ
    email         TEXT         NOT NULL UNIQUE,
    -- bcrypt のハッシュ（60文字固定）。TEXT にしておけば将来 argon2 に替えても入る
    password_hash TEXT         NOT NULL,
    -- ロールは2値のみ。ENUM 型ではなく TEXT + CHECK にしている（理由は下の解説）
    role          TEXT         NOT NULL DEFAULT 'user'
                               CHECK (role IN ('user', 'admin')),
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- ============================================================
-- refresh_tokens : リフレッシュトークンの実体。行を消すことが失効を意味する
-- ============================================================
CREATE TABLE refresh_tokens (
    id         BIGSERIAL   PRIMARY KEY,
    -- ユーザーが消えたらトークンも消す。CASCADE が妥当な数少ないケース
    user_id    BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    -- SHA-256 の hex 文字列（64文字）。bcrypt ではない（理由は前回の設計メモの通り）
    -- UNIQUE にすることで、このカラム1本で検索できる＝索引も兼ねる
    token_hash TEXT        NOT NULL UNIQUE,
    -- 有効期限。期限切れ行の掃除はアプリ側の責務（今回は起動時 or 手動で十分）
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 「あるユーザーの全トークンを失効させる」= user_id で DELETE するため索引を張る
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens (user_id);

-- ============================================================
-- products : 商品マスタ。第4層（楽観ロック）の対象
-- ============================================================
CREATE TABLE products (
    id         BIGSERIAL   PRIMARY KEY,
    -- 業務上の一意キー。id とは別に人間が読める識別子を持つのは実務では普通
    sku        TEXT        NOT NULL UNIQUE,
    name       TEXT        NOT NULL,
    -- 金額は「円」を整数で持つ。float は丸め誤差が出るので金額には使わない。
    -- 小数通貨や税計算が要るなら NUMERIC だが、今回は円のみなので INTEGER で足りる
    price      INTEGER     NOT NULL CHECK (price >= 0),
    -- 楽観ロック用。UPDATE のたびに +1 する。アプリが更新する（トリガーは使わない）
    version    INTEGER     NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================
-- inventories : 在庫。第0〜3層の主戦場。products と 1:1
-- ============================================================
CREATE TABLE inventories (
    -- product_id をそのまま PRIMARY KEY にすることで、
    -- 「1商品につき在庫行は必ず1つ」を DB レベルで強制する（別に id 列を持たない）
    product_id        BIGINT      PRIMARY KEY REFERENCES products(id) ON DELETE CASCADE,
    -- 物理在庫。倉庫に実在する数
    quantity_on_hand  INTEGER     NOT NULL DEFAULT 0 CHECK (quantity_on_hand >= 0),
    -- 引き当て済み。注文されたがまだ出荷していない数
    quantity_reserved INTEGER     NOT NULL DEFAULT 0 CHECK (quantity_reserved >= 0),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- 不変条件：引き当ては物理在庫を超えない。
    -- 複数列にまたがる CHECK は「文の終わり」に評価されるので、
    -- 1つの UPDATE で両方を減らす場合、途中経過が一瞬崩れても問題にならない
    CONSTRAINT chk_reserved_le_on_hand CHECK (quantity_reserved <= quantity_on_hand)
);

-- ============================================================
-- orders : 受注ヘッダ。ステータス遷移とトランザクション境界の中心
-- ============================================================
CREATE TABLE orders (
    id         BIGSERIAL   PRIMARY KEY,
    -- RESTRICT：注文履歴が残っているユーザーは消せない。
    -- users 側を CASCADE にすると受注データが消し飛ぶので、ここは意図的に RESTRICT
    user_id    BIGINT      NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    -- 遷移は pending → shipped、pending → cancelled の2本だけ。
    -- 「どの遷移が許されるか」は CHECK では表現できないのでアプリ（service層）の責務
    status     TEXT        NOT NULL DEFAULT 'pending'
                           CHECK (status IN ('pending', 'shipped', 'cancelled')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 「自分の注文一覧」を引くため。認可チェックでも user_id を使う
CREATE INDEX idx_orders_user_id ON orders (user_id);

-- ============================================================
-- order_items : 受注明細。ロック順序ソート（第2層）の単位
-- ============================================================
CREATE TABLE order_items (
    id         BIGSERIAL PRIMARY KEY,
    -- 注文を消したら明細も消える。明細は注文に完全従属するので CASCADE が妥当
    order_id   BIGINT    NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    -- 注文実績のある商品は消せない
    product_id BIGINT    NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    -- 0個の明細は無意味なので > 0（>= 0 ではない）
    quantity   INTEGER   NOT NULL CHECK (quantity > 0),
    -- 注文時点の価格を焼き付ける。products.price はあとで変わるので参照では駄目
    unit_price INTEGER   NOT NULL CHECK (unit_price >= 0),
    -- 同一注文内で同じ商品が2行に分かれないよう強制する。
    -- これにより第2層のデッドロック回避が「product_id で並べ替えるだけ」で成立する
    CONSTRAINT uq_order_items_order_product UNIQUE (order_id, product_id)
);

-- 注文詳細取得時に order_id で引くため
CREATE INDEX idx_order_items_order_id ON order_items (order_id);