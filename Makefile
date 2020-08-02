DOCKER_IMAGE_TAG=latest

image:
	DOCKER_BUILDKIT=1 docker build -t ablyrealtime/ably-boomer:$(DOCKER_IMAGE_TAG) .

push:
	docker push ablyrealtime/ably-boomer:$(DOCKER_IMAGE_TAG)

build:
	go vet ./ably
	go build -o ably-boomer ./...

cover:
	mkdir -p ./coverage
	go test -covermode=atomic -coverprofile=coverage/coverage.out ./ably ./ably/perf
	go tool cover -html=./coverage/coverage.out -o=./coverage/coverage.html


test:
	go test ./ably ./ably/perf

fmt:
	go fmt ./ably

.PHONY: image push build test fmt
