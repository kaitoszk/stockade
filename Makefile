include .env
# 以降に定義した変数も含め、全部を子プロセスの環境変数として渡す
export

# .env の値から組み立てる。手書きの重複を排除
DATABASE_URL := postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@localhost:$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable

.PHONY: up down reset migrate-up migrate-down migrate-create migrate-version psql

up:
	docker compose up -d

down:
	docker compose down

reset:
	docker compose down -v
	docker compose up -d

migrate-up:
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path migrations -database "$(DATABASE_URL)" down 1

migrate-create:
	migrate create -ext sql -dir migrations -seq $(NAME)

migrate-version:
	migrate -path migrations -database "$(DATABASE_URL)" version

psql:
	docker compose exec db psql -U $(POSTGRES_USER) -d $(POSTGRES_DB)