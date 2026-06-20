FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o /app/auth-service .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/auth-service /app/auth-service
CMD ["/app/auth-service"]