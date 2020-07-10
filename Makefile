DOCKER_IMAGE_TAG=latest

image:
	DOCKER_BUILDKIT=1 docker build -t ably-boomer:$(DOCKER_IMAGE_TAG) .

build:
	go vet ./ably
	go build -o ably-boomer ./...

test:
	go test ./ably

fmt:
	go fmt ./ably

.PHONY: image build test fmt
