DOCKER_IMAGE_REPO=ablyrealtime/ably-boomer
DOCKER_IMAGE_TAG=latest

all: build

# install shared tools to local bin directory
BIN = $(CURDIR)/bin
$(BIN):
	@mkdir -p $@
$(BIN)/%: | $(BIN)
	@tmp=$$(mktemp -d); \
		env GO111MODULE=off GOPATH=$$tmp GOBIN=$(BIN) go get $(PACKAGE) \
		|| ret=$$?; \
		rm -rf $$tmp ; exit $$ret
$(BIN)/golint: PACKAGE=golang.org/x/lint/golint

image:
	DOCKER_BUILDKIT=1 docker build -t $(DOCKER_IMAGE_REPO):$(DOCKER_IMAGE_TAG) .

push:
	docker push $(DOCKER_IMAGE_REPO):$(DOCKER_IMAGE_TAG)

build:
	go vet ./cmd/ably-boomer
	go build -o ./bin/ably-boomer ./cmd/ably-boomer

cover: lint
	mkdir -p ./coverage
	go test -covermode=atomic -coverprofile=coverage/coverage.out ./...
	go tool cover -html=./coverage/coverage.out -o=./coverage/coverage.html

GOLINT = $(BIN)/golint
lint: | $(GOLINT)
	$(GOLINT) -set_exit_status ./...

test: fmt
	go test ./...

fmt:
	go fmt ./...

.PHONY: all image push build cover lint test fmt
