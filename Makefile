.PHONY: up up-all down restart ps logs build vet test lint swag gen-keys ocr-benchmark test-ocr e2e test-pkg test-auth test-gateway test-org test-identity test-notification test-admission test-finance test-exam test-scoring test-selection test-student test-nim demo-seed demo-reset backup restore

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
	go build ./pkg/... ./services/auth-service/... ./services/api-gateway/... ./services/organization-service/... ./services/identity-service/... ./services/notification-service/... ./services/admission-service/... ./services/finance-service/... ./services/exam-service/... ./services/scoring-service/... ./services/selection-service/... ./services/student-service/... ./services/nim-service/...

vet:
	go vet ./pkg/... ./services/auth-service/... ./services/api-gateway/... ./services/organization-service/... ./services/identity-service/... ./services/notification-service/... ./services/admission-service/... ./services/finance-service/... ./services/exam-service/... ./services/scoring-service/... ./services/selection-service/... ./services/student-service/... ./services/nim-service/...

test:
	go test -v ./pkg/... ./services/auth-service/... ./services/api-gateway/... ./services/organization-service/... ./services/identity-service/... ./services/notification-service/... ./services/admission-service/... ./services/finance-service/... ./services/exam-service/... ./services/scoring-service/... ./services/selection-service/... ./services/student-service/... ./services/nim-service/...

lint:
	golangci-lint run ./pkg/... ./services/auth-service/... ./services/api-gateway/... ./services/organization-service/... ./services/identity-service/... ./services/notification-service/... ./services/admission-service/... ./services/finance-service/... ./services/exam-service/... ./services/scoring-service/... ./services/selection-service/... ./services/student-service/... ./services/nim-service/...

e2e:
	bash scripts/e2e.sh

# Target pengujian per-service (sesuai PRD Bagian 6)
test-pkg:
	go test -v ./pkg/...
test-auth:
	go test -v ./services/auth-service/...
test-gateway:
	go test -v ./services/api-gateway/...
test-org:
	go test -v ./services/organization-service/...
test-identity:
	go test -v ./services/identity-service/...
test-notification:
	go test -v ./services/notification-service/...
test-admission:
	go test -v ./services/admission-service/...
test-finance:
	go test -v ./services/finance-service/...
test-exam:
	go test -v ./services/exam-service/...
test-scoring:
	go test -v ./services/scoring-service/...
test-selection:
	go test -v ./services/selection-service/...
test-student:
	go test -v ./services/student-service/...
test-nim:
	go test -v ./services/nim-service/...

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

demo-seed:
	@echo "Seeding initial demo master data..."
	docker compose -f $(COMPOSE_FILE) exec -T auth-service /app/auth-service -seed || true
	docker compose -f $(COMPOSE_FILE) exec -T admission-service /app/admission-service -seed || true
	docker compose -f $(COMPOSE_FILE) exec -T finance-service /app/finance-service -seed || true
	@echo "Demo data seeding completed."

demo-reset:
	@echo "Resetting demo transaction data..."
	docker compose -f $(COMPOSE_FILE) exec -T postgres psql -U pmb_user -d pmb_admission -c "TRUNCATE applications, application_documents, selection_results CASCADE;" || true
	docker compose -f $(COMPOSE_FILE) exec -T postgres psql -U pmb_user -d pmb_finance -c "TRUNCATE invoices, payments, receipts, installment_requests, installments CASCADE;" || true
	@echo "Demo transaction data reset completed."

backup:
	@echo "Backing up databases..."
	@mkdir -p backups
	docker compose -f $(COMPOSE_FILE) exec -T postgres pg_dumpall -U pmb_user | gzip > backups/pmb_backup_$$(date +%Y%m%d_%H%M%S).sql.gz
	@echo "Database backup completed in backups/."

restore:
	@echo "Restoring database from latest backup in backups/..."
	@LATEST_BACKUP=$$(ls -t backups/*.sql.gz 2>/dev/null | head -n 1); \
	if [ -n "$$LATEST_BACKUP" ]; then \
		echo "Restoring from $$LATEST_BACKUP"; \
		gunzip -c "$$LATEST_BACKUP" | docker compose -f $(COMPOSE_FILE) exec -T postgres psql -U pmb_user -d postgres; \
		echo "Restore completed successfully."; \
	else \
		echo "No backup file found in backups/"; \
	fi

