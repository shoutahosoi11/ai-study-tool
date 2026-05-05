.PHONY: db migrate backend worker dev

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

worker:
	cd backend && go run ./cmd/question-worker/...

dev: db
	@echo "==> Waiting for postgres..."
	@until docker-compose exec -T postgres pg_isready -U postgres -d ai_study_tool 2>/dev/null; do sleep 1; done
	$(MAKE) migrate
	$(MAKE) backend
