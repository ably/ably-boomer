image:
	docker build .

build:
	go vet
	go build

.PHONY: image build
