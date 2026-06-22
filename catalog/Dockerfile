# syntax=docker/dockerfile:1
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/root/go/pkg/mod \
    go mod download
COPY . .
RUN --mount=type=cache,target=/root/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -o /app/catalog-service .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/catalog-service /app/catalog-service
COPY --from=builder /app/config /app/config
CMD ["/app/catalog-service"]
