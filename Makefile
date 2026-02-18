.PHONY: run build test test-unit test-integration migrate-up migrate-down generate-keys tidy docker-up docker-down

# ── Go ─────────────────────────────────────────────────────────────────────────
run:
	go run ./cmd/api/main.go

build:
	CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o bin/api ./cmd/api/main.go

tidy:
	go mod tidy

# ── Tests ──────────────────────────────────────────────────────────────────────
test-unit:
	go test -short -race -count=1 ./pkg/... ./internal/...

test-integration:
	go test -race -count=1 -timeout 120s ./tests/integration/...

test: test-unit test-integration

# ── Migrations ─────────────────────────────────────────────────────────────────
MIGRATE_DSN ?= "mysql://$(DB_USER):$(DB_PASSWORD)@tcp($(DB_HOST):$(DB_PORT))/$(DB_NAME)"

migrate-up:
	migrate -path migrations -database $(MIGRATE_DSN) up

migrate-down:
	migrate -path migrations -database $(MIGRATE_DSN) down

migrate-create:
	@read -p "Migration name: " name; \
	migrate create -ext sql -dir migrations -seq $$name

# ── RSA Key Generation ─────────────────────────────────────────────────────────
generate-keys:
	@echo "Generating RSA-2048 key pair..."
	openssl genrsa -out private.pem 2048
	openssl rsa -in private.pem -pubout -out public.pem
	@echo "Keys written to private.pem and public.pem"
	@echo "Base64 private key (for JWT_PRIVATE_KEY_B64):"
	@base64 -i private.pem
	@echo ""
	@echo "Base64 public key (for JWT_PUBLIC_KEY_B64):"
	@base64 -i public.pem

# ── Docker ─────────────────────────────────────────────────────────────────────
docker-up:
	docker-compose up --build -d

docker-down:
	docker-compose down -v

docker-logs:
	docker-compose logs -f app

# ── Lint ───────────────────────────────────────────────────────────────────────
lint:
	golangci-lint run ./...
