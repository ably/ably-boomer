DOCKER_IMAGE_TAG=latest

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
	DOCKER_BUILDKIT=1 docker build -t ablyrealtime/ably-boomer:$(DOCKER_IMAGE_TAG) .

push:
	docker push ablyrealtime/ably-boomer:$(DOCKER_IMAGE_TAG)

build:
	go vet ./ably
	go build -o ably-boomer ./...

cover: lint
	mkdir -p ./coverage
	go test -covermode=atomic -coverprofile=coverage/coverage.out ./ably/...
	go tool cover -html=./coverage/coverage.out -o=./coverage/coverage.html

GOLINT = $(BIN)/golint
lint: | $(GOLINT)
	$(GOLINT) -set_exit_status ./ably/...

test: fmt lint
	go test ./ably/...

fmt:
	go fmt ./ably/...

.PHONY: image push build test fmt
