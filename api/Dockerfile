FROM golang:1.26-alpine AS builder
WORKDIR /repo
COPY . .
RUN cd api && go build -o /server .

FROM alpine:3.20
COPY --from=builder /server /server
ENTRYPOINT ["/server"]
