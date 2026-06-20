FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o /app/inventory-service .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/inventory-service /app/inventory-service
CMD ["/app/inventory-service"]