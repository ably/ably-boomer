DOCKER_IMAGE_TAG=latest

image:
	DOCKER_BUILDKIT=1 docker build -t ablyrealtime/ably-boomer:$(DOCKER_IMAGE_TAG) .

push:
	docker push ablyrealtime/ably-boomer:$(DOCKER_IMAGE_TAG)

build:
	go vet ./ably
	go build -o ably-boomer ./...

test:
	go test ./ably

fmt:
	go fmt ./ably

.PHONY: image push build test fmt
