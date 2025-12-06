FROM golang:1.25.4-alpine

WORKDIR /app

RUN go install github.com/air-verse/air@latest

COPY go.mod go.sum ./
RUN go mod download

COPY backend backend

EXPOSE 8080

CMD ["air", "-c", "backend/.air.toml"]