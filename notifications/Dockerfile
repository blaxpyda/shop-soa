FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o /app/notifications-service .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/notifications-service /app/notifications-service
CMD ["/app/notifications-service"]