FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o /app/payments-service .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/payments-service /app/payments-service
CMD ["/app/payments-service"]