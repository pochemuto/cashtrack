FROM node:20-alpine AS ui-builder

WORKDIR /ui

COPY client/package*.json ./
RUN npm install

COPY client .
RUN npm run build



FROM golang:1.25.4-alpine AS go-builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY server server
COPY --from=ui-builder /ui/build ./public

RUN CGO_ENABLED=0 GOOS=linux go build -o app ./server/cmd/server



FROM alpine:3.20

WORKDIR /app

COPY --from=go-builder /app/app .
COPY --from=go-builder /app/public ./public

EXPOSE 8080

CMD ["./app"]