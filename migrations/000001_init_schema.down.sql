-- 外部キーの参照関係があるため、子テーブルから先に落とす。
-- （CASCADE を付ければ順不同でもいけるが、意図しないテーブルまで落ちるので使わない）
DROP TABLE IF EXISTS order_items;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS inventories;
DROP TABLE IF EXISTS products;
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS users;