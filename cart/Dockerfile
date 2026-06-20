FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o /app/cart-service .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/cart-service /app/cart-service
CMD ["/app/cart-service"]