DOCKER_IMAGE_TAG=latest

image:
	DOCKER_BUILDKIT=1 docker build -t ably/ably-boomer:$(DOCKER_IMAGE_TAG) .

build:
	go vet
	go build

.PHONY: image build
