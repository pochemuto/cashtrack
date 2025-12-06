CODEGEN_IMAGE=cashtrack-codegen

server:
	go run server/cmd/server/main.go

client:
	cd client && npm run dev

generate:
	docker compose -f docker/docker-compose.generate.yml up generator

dev-local-deps:
	docker compose -f docker/docker-compose.dev.yml up -d db

dev-local:
	make dev-local-deps
	make -j2 server client

dev:
	docker compose -f docker/docker-compose.dev.yml up --build --abort-on-container-exit

prod:
	docker compose -f docker/docker-compose.yml up --build

.PHONY: generate server client