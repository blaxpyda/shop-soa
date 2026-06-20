FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o /app/transactions-service .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/transactions-service /app/transactions-service
CMD ["/app/transactions-service"]