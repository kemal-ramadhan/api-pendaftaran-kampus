.PHONY: up down restart ps logs build vet test lint swag gen-keys ocr-benchmark test-ocr

COMPOSE_FILE := deploy/docker-compose.yml

up:
	docker compose -f $(COMPOSE_FILE) up -d postgres redis mailpit minio minio-init

up-all:
	docker compose -f $(COMPOSE_FILE) up -d

down:
	docker compose -f $(COMPOSE_FILE) down

restart: down up

ps:
	docker compose -f $(COMPOSE_FILE) ps

logs:
	docker compose -f $(COMPOSE_FILE) logs -f

build:
	go build ./pkg/... ./services/auth-service/... ./services/api-gateway/... ./services/organization-service/...

vet:
	go vet ./pkg/... ./services/auth-service/... ./services/api-gateway/... ./services/organization-service/...

test:
	go test -v ./pkg/... ./services/auth-service/... ./services/api-gateway/... ./services/organization-service/...

lint:
	golangci-lint run ./pkg/... ./services/auth-service/... ./services/api-gateway/...

swag:
	@echo "Generating swagger docs..."
	@if command -v swag >/dev/null 2>&1; then \
		cd services/auth-service && swag init -g cmd/main.go -o docs; \
	else \
		echo "swag CLI not found. Install with: go install github.com/swaggo/swag/cmd/swag@latest"; \
	fi

gen-keys:
	go run ./scripts/genkeys/main.go

ocr-benchmark:
	docker compose -f $(COMPOSE_FILE) exec -T ocr-service python tests/benchmark.py
	docker compose -f $(COMPOSE_FILE) cp ocr-service:/app/ocr-benchmark.md docs/ocr-benchmark.md

test-ocr:
	docker compose -f $(COMPOSE_FILE) exec -T ocr-service pytest -v tests/

