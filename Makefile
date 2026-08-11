BINARY := agent-stripe
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)
GOCACHE ?= $(CURDIR)/.cache/go-build

build:
	GOCACHE=$(GOCACHE) go build -buildvcs=false -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/agent-stripe

build-mock:
	GOCACHE=$(GOCACHE) go build -buildvcs=false -o mockstripe ./cmd/mockstripe

mock:
	GOCACHE=$(GOCACHE) go run ./cmd/mockstripe

mock-dev:
	AGENT_STRIPE_BASE_URL=http://127.0.0.1:12111 STRIPE_API_KEY=sk_test_mock GOCACHE=$(GOCACHE) go run ./cmd/agent-stripe $(ARGS)

test:
	GOCACHE=$(GOCACHE) go test ./... -count=1

test-short:
	GOCACHE=$(GOCACHE) go test ./... -count=1 -short

lint:
	golangci-lint run ./...

# Validates the requests the CLI emits against Stripe's published OpenAPI spec:
# every /v1 path must exist and every query parameter must be one the endpoint
# declares. Downloads the spec (~8MB) into .cache on first run. Stripe does not
# publish the /v2 namespace in these files, so those requests are listed for
# review rather than checked — see design-docs/accounts-v2.md.
STRIPE_SPEC := $(CURDIR)/.cache/stripe-openapi/spec3.json

$(STRIPE_SPEC):
	@mkdir -p $(dir $(STRIPE_SPEC))
	curl -fsSL -o $(STRIPE_SPEC) https://raw.githubusercontent.com/stripe/openapi/master/openapi/spec3.json

apicheck: $(STRIPE_SPEC)
	STRIPE_OPENAPI_SPEC=$(STRIPE_SPEC) GOCACHE=$(GOCACHE) go test -tags apicheck ./internal/apicheck/ -count=1 -v

apicheck-refresh:
	rm -f $(STRIPE_SPEC)
	$(MAKE) apicheck

# Scoped to tracked files: this repo keeps a module cache under .cache/, which
# the go tool skips (dot-directory) but gofmt and goimports walk into, so a bare
# `-w .` rewrites vendored dependencies and makes `gofmt -l .` report noise.
fmt:
	gofmt -w $$(git ls-files '*.go')
	@command -v goimports >/dev/null && goimports -w $$(git ls-files '*.go') || echo "goimports not installed (optional; install: go install golang.org/x/tools/cmd/goimports@latest)"

clean:
	rm -f $(BINARY)
	rm -f mockstripe
	rm -rf dist/

dev:
	GOCACHE=$(GOCACHE) go run ./cmd/agent-stripe $(ARGS)

vet:
	GOCACHE=$(GOCACHE) go vet ./...

.PHONY: build build-mock mock mock-dev test test-short lint apicheck apicheck-refresh fmt clean dev vet
