CODEGEN_IMAGE=cashtrack-codegen

build-generate-image:
	docker build -f Dockerfile.generate.dev -t $(CODEGEN_IMAGE) .

run-generate:
	docker run --rm \
	  --mount type=bind,src=.,dst=/app \
	  -w /app \
	$(CODEGEN_IMAGE) \
	buf generate && wire ./...

server:
	go run server/cmd/server/main.go

client:
	cd client && npm run dev

generate: build-generate-image run-generate

dev-local:
	make -j2 server client

.PHONY: generate server client build-generate-image run-generate