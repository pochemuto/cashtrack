FROM node:20-alpine AS frontend-builder

WORKDIR /frontend

COPY frontend/package*.json ./
RUN npm install

COPY frontend .
RUN npm run build



FROM golang:1.25.4-alpine AS backend-builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY backend backend
COPY --from=frontend-builder /frontend/build ./public

ARG TARGETOS
ARG TARGETARCH
ENV CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH
RUN go build -o app ./backend/cmd/server



FROM alpine:3.20

WORKDIR /app

COPY --from=backend-builder /app/app .
COPY --from=backend-builder /app/public ./public

EXPOSE 8080

CMD ["./app"]