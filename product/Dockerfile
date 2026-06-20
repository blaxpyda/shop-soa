FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o /app/product-service .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/product-service /app/product-service
CMD ["/app/product-service"]