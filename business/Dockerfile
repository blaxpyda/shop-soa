# syntax=docker/dockerfile:1
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/root/go/pkg/mod \
    go mod download
COPY . .
RUN --mount=type=cache,target=/root/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -o /app/business-service .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/business-service /app/business-service
COPY --from=builder /app/config /app/config
CMD ["/app/business-service"]
