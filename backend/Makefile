GOLANGCI_LINT_VERSION := v2.15.0
GOLANGCI_LINT := $(CURDIR)/bin/golangci-lint

.PHONY: run build test lint lint-fix tools

run:
	go run ./cmd/api

build:
	go build -o bin/api ./cmd/api

test:
	go test -race -count=1 ./...

lint: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run ./...

lint-fix: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run --fix ./...

tools: $(GOLANGCI_LINT)

$(GOLANGCI_LINT):
	GOBIN=$(CURDIR)/bin go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
