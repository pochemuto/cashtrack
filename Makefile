COMPOSE_DEV=docker/docker-compose.dev.yml
COMPOSE_GEN=docker/docker-compose.generate.yml
COMPOSE_PROD=docker/docker-compose.yml
COMPOSE_BUILD=docker/docker-compose.build.yml
IMAGE=pochemuto/cashtrack
TAG=latest

backend:
	go run backend/cmd/server/main.go

frontend:
	cd frontend && npm run dev

generate:
	docker compose -f $(COMPOSE_GEN) -p cashtrack-gen up generator --build

dev-local-deps:
	docker compose -f $(COMPOSE_DEV) -p cashtrack-dev-local up -d db

dev-local:
	$(MAKE) dev-local-deps
	$(MAKE) -j2 backend frontend

dev:
	docker compose -f $(COMPOSE_DEV) -p cashtrack-dev up --build --abort-on-container-exit

build:
	docker buildx build \
		--platform linux/amd64,linux/arm64 \
		-t $(IMAGE):$(TAG) \
		-f docker/Dockerfile \
		--push \
		.

push:
	docker compose -f $(COMPOSE_PROD) -f $(COMPOSE_BUILD) push

deploy:
	docker --context $(context) compose -f $(COMPOSE_PROD) pull
	docker --context $(context) compose -f $(COMPOSE_PROD) -p cashtrack up -d


new-migration:
	@if [ -z "$(name)" ]; then echo "Usage: make migrate name=migration_name"; exit 1; fi
	docker compose -f $(COMPOSE_GEN) -p cashtrack-gen run --build --rm generator \
	  sh -c "goose -s create $(name) sql"

generate-fresh:
	docker compose -f $(COMPOSE_GEN) -p cashtrack-gen down
	$(MAKE) generate

.PHONY: backend frontend generate dev dev-local dev-local-deps prod migrate