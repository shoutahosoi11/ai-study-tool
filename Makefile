.PHONY: db migrate backend test test-backend test-backend-race test-frontend test-mobile dev

DB_URL ?= postgres://postgres:postgres@localhost:5432/ai_study_tool?sslmode=disable

db:
	docker-compose up -d postgres

migrate:
	@echo "==> Running migrations..."
	@for f in $$(ls backend/db/migrations/*.sql | sort); do \
		echo "  $$f"; \
		psql "$(DB_URL)" -q -f "$$f" 2>&1 | grep -v "^$$" | grep -v "already exists" || true; \
	done
	@echo "==> Done."

backend:
	cd backend && go run ./cmd/main.go

test: test-backend test-frontend test-mobile

test-backend:
	cd backend && go test ./... && go build ./...

test-backend-race:
	cd backend && go test -race ./internal/usecase ./internal/middleware ./internal/infrastructure/cloudtasks

test-frontend:
	cd frontend && npm test

test-mobile:
	cd mobile && npm test

dev: db
	@echo "==> Waiting for postgres..."
	@until docker-compose exec -T postgres pg_isready -U postgres -d ai_study_tool 2>/dev/null; do sleep 1; done
	$(MAKE) migrate
	$(MAKE) backend
