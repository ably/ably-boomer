# syntax=docker/dockerfile:experimental
FROM golang:1.14.4-alpine3.12 AS builder

WORKDIR /opt/ably

RUN apk add --no-cache --upgrade make gcc libc-dev

COPY . .

RUN \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go \
    make build

FROM alpine:3.12

RUN addgroup -S ably && adduser -S ably -G ably

WORKDIR /opt/ably

USER ably

COPY --from=builder /opt/ably/ably-boomer /opt/ably/ably-boomer

ENTRYPOINT ["./ably-boomer"]
