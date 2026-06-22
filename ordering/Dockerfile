# syntax=docker/dockerfile:1
# Build context must be the repo root so the catalog replace directive resolves.
FROM golang:1.26-alpine AS builder
WORKDIR /repo
COPY . .
RUN --mount=type=cache,target=/root/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    cd ordering && go build -o /ordering-service .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /ordering-service /app/ordering-service
COPY --from=builder /repo/ordering/config /app/config
EXPOSE 50054
CMD ["/app/ordering-service"]
