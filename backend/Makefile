GOLANGCI_LINT_VERSION := v2.15.0
GOLANGCI_LINT := $(CURDIR)/bin/golangci-lint

COMPOSE := docker compose -f ../docker-compose.yml

.PHONY: run build test test-integration lint lint-fix tools infra-up infra-down migrate-up

infra-up:
	$(COMPOSE) up -d --wait postgres redis
	$(COMPOSE) run --rm migrate

infra-down:
	$(COMPOSE) down

migrate-up:
	$(COMPOSE) run --rm migrate

run:
	go run ./cmd/api

build:
	go build -o bin/api ./cmd/api

test:
	go test -race -count=1 ./...

test-integration:
	go test -race -count=1 -tags integration ./...

lint: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run ./...

lint-fix: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run --fix ./...

tools: $(GOLANGCI_LINT)

$(GOLANGCI_LINT):
	GOBIN=$(CURDIR)/bin go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
