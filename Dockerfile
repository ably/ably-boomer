FROM golang:1.14.4-alpine3.12 AS builder

WORKDIR /opt/ably

RUN apk add --no-cache --upgrade make gcc libc-dev

COPY . .

RUN make build



FROM golang:1.14.4-alpine3.12

WORKDIR /opt/ably

COPY --from=builder /opt/ably/ably-boomer /opt/ably/ably-boomer

ENTRYPOINT ["./ably-boomer"]
