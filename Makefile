BINARY := agent-stripe
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/agent-stripe

build-mock:
	go build -o mockstripe ./cmd/mockstripe

test:
	go test ./... -count=1

test-short:
	go test ./... -count=1 -short

lint:
	golangci-lint run ./...

fmt:
	gofmt -w .
	goimports -w .

clean:
	rm -f $(BINARY)
	rm -f mockstripe
	rm -rf dist/

dev:
	go run ./cmd/agent-stripe $(ARGS)

vet:
	go vet ./...

.PHONY: build build-mock test test-short lint fmt clean dev vet
