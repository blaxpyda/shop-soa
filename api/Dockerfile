# syntax=docker/dockerfile:1
FROM golang:1.26-alpine AS builder
WORKDIR /repo
COPY . .
RUN --mount=type=cache,target=/root/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    cd api && go build -o /server .

FROM alpine:3.20
COPY --from=builder /server /server
ENTRYPOINT ["/server"]
