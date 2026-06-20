FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o /app/orders-service .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/orders-service /app/orders-service
CMD ["/app/orders-service"]