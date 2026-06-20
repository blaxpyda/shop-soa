FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o /app/business-service .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/business-service /app/business-service
CMD ["/app/business-service"]