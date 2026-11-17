GO ?= go
BIN := bin/stirling-pdf

.PHONY: all build test test-race e2e cover lint generate check-generate clean

all: test build

build:
	$(GO) build -o $(BIN) ./cmd/stirling-pdf

test:
	$(GO) test ./...

test-race:
	$(GO) test -race ./...

e2e:
	$(GO) test -tags=e2e -count=1 -timeout 15m ./e2e -v

cover:
	$(GO) test -coverprofile=coverage.txt ./...
	$(GO) tool cover -html=coverage.txt -o coverage.html

lint:
	@test -z "$$(gofmt -l $$(find . -name '*.go' -not -path './vendor/*'))" || (echo "gofmt required:"; gofmt -l $$(find . -name '*.go' -not -path './vendor/*'); exit 1)
	$(GO) vet ./...

generate:
	$(GO) generate ./api ./internal/ops
	gofmt -w api internal/ops cmd stirling

check-generate:
	$(GO) generate ./api ./internal/ops
	gofmt -w api internal/ops
	git diff --exit-code -- api/client.gen.go api/types.gen.go internal/ops/registry.gen.go docs/operations.md

clean:
	rm -rf bin coverage.txt coverage.html
