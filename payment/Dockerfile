# syntax=docker/dockerfile:1
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/root/go/pkg/mod \
    go mod download
COPY . .
RUN --mount=type=cache,target=/root/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -o /app/payment-service .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/payment-service /app/payment-service
COPY --from=builder /app/config /app/config
EXPOSE 50055 8081
CMD ["/app/payment-service"]
