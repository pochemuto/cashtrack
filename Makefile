COMPOSE_DEV=docker/docker-compose.dev.yml
COMPOSE_GEN=docker/docker-compose.generate.yml
COMPOSE_PROD=docker/docker-compose.yml
IMAGE=pochemuto/cashtrack
TAG=latest

backend:
	go run backend/cmd/server/main.go

frontend:
	cd frontend && npm run dev

generate:
	docker compose -f $(COMPOSE_GEN) -p cashtrack-gen up generator --build

development-local-deps:
	docker compose -f $(COMPOSE_DEV) -p cashtrack-dev-local --profile localhost up -d db migrate

development-local:
	$(MAKE) development-local-deps
	$(MAKE) -j2 backend frontend

development:
	docker compose -f $(COMPOSE_DEV) --env-file .env.development -p cashtrack-dev up --build

push:
	docker buildx build \
		--platform linux/amd64,linux/arm64 \
		-t $(IMAGE):$(TAG) \
		-f docker/Dockerfile \
		--push \
		.

deploy:
	@if [ -z "$(context)" ]; then echo "Usage: make deploy context=docker_context"; exit 1; fi
	docker --context $(context) compose -f $(COMPOSE_PROD) pull
	docker --context $(context) compose -f $(COMPOSE_PROD) -p cashtrack up -d


new-migration:
	@if [ -z "$(name)" ]; then echo "Usage: make new-migration name=migration_name"; exit 1; fi
	docker compose -f $(COMPOSE_GEN) -p cashtrack-gen run --build --rm generator \
	  sh -c "goose -s create $(name) sql"

generate-fresh:
	docker compose -f $(COMPOSE_GEN) -p cashtrack-gen down
	$(MAKE) generate

# Run migrations locally against localhost Postgres (dev-local scenario)
migrate-local:
	docker compose -f $(COMPOSE_GEN) -p cashtrack-gen run --build --rm generator \
	  sh -c 'GOOSE_DRIVER=postgres GOOSE_DBSTRING="postgresql://cashtrack:cashtrack@localhost:25432/cashtrack?sslmode=disable" GOOSE_MIGRATION_DIR=./db/migrations goose up'

.PHONY: backend frontend generate development development-local development-local-deps push deploy new-migration generate-fresh migrate-local