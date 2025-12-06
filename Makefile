CODEGEN_IMAGE=cashtrack-codegen

backend:
	go run backend/cmd/server/main.go

frontend:
	cd frontend && npm run dev

generate:
	docker compose -f docker/docker-compose.generate.yml -p cashtrack-gen up generator

dev-local-deps:
	docker compose -f docker/docker-compose.dev.yml -p cashtrack-dev-local up -d db

dev-local:
	make dev-local-deps
	make -j2 backend frontend

dev:
	docker compose -f docker/docker-compose.dev.yml -p cashtrack-dev up --build --abort-on-container-exit

prod:
	docker compose -f docker/docker-compose.yml -p cashtrack up --build

.PHONY: generate backend frontend dev prod dev-local