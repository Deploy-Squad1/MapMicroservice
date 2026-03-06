FROM golang:1.26-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o map-service ./cmd/api

FROM alpine:3.23
WORKDIR /app

COPY --from=builder /app/map-service .

COPY --from=builder /app/migrations ./migrations

EXPOSE 8080

ENTRYPOINT ["./map-service"]