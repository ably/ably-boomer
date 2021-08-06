# syntax=docker/dockerfile:experimental
FROM golang:1.16-alpine3.13 AS builder

WORKDIR /home/ablyboomer

RUN apk add --no-cache --upgrade make gcc libc-dev

COPY . .

RUN \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go \
    make build

FROM alpine:3.13

RUN adduser -S ablyboomer
USER ablyboomer
WORKDIR /home/ablyboomer

COPY --from=builder /home/ablyboomer/bin/ably-boomer /bin/ably-boomer

ENTRYPOINT ["/bin/ably-boomer"]
