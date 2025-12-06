COMPOSE_DEV=docker/docker-compose.dev.yml
COMPOSE_GEN=docker/docker-compose.generate.yml
COMPOSE_PROD=docker/docker-compose.yml

backend:
	go run backend/cmd/server/main.go

frontend:
	cd frontend && npm run dev

generate:
	docker compose -f $(COMPOSE_GEN) -p cashtrack-gen up generator

dev-local-deps:
	docker compose -f $(COMPOSE_DEV) -p cashtrack-dev-local up -d db

dev-local:
	$(MAKE) dev-local-deps
	$(MAKE) -j2 backend frontend

dev:
	docker compose -f $(COMPOSE_DEV) -p cashtrack-dev up --build --abort-on-container-exit

prod:
	docker compose -f $(COMPOSE_PROD) -p cashtrack up --build

migrate:
	@if [ -z "$(name)" ]; then echo "Usage: make migrate name=migration_name"; exit 1; fi
	docker compose -f $(COMPOSE_GEN) -p cashtrack-gen run --build --rm generator \
	  sh -c "goose -s create $(name) sql"

generate-fresh:
	docker compose -f $(COMPOSE_GEN) -p cashtrack-gen down
	$(MAKE) generate

.PHONY: backend frontend generate dev dev-local dev-local-deps prod migrate