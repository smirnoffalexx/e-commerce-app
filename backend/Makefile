GOLANGCI_LINT_VERSION := v2.15.0
GOLANGCI_LINT := $(CURDIR)/bin/golangci-lint
MOCKERY_VERSION := v3.8.1
MOCKERY := $(CURDIR)/bin/mockery
SWAG_VERSION := v1.16.6
SWAG := $(CURDIR)/bin/swag

COMPOSE := docker compose -f ../docker-compose.yml

-include ../.env
export

.PHONY: run build test test-integration lint lint-fix mocks swagger tools infra-up infra-down migrate-up

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

mocks: $(MOCKERY)
	$(MOCKERY)

swagger: $(SWAG)
	$(SWAG) fmt -d ./internal/delivery/http
	$(SWAG) init -g internal/delivery/http/router.go -o docs --outputTypes go,yaml

tools: $(GOLANGCI_LINT) $(MOCKERY) $(SWAG)

$(SWAG):
	GOBIN=$(CURDIR)/bin go install github.com/swaggo/swag/cmd/swag@$(SWAG_VERSION)

$(MOCKERY):
	GOBIN=$(CURDIR)/bin go install github.com/vektra/mockery/v3@$(MOCKERY_VERSION)

$(GOLANGCI_LINT):
	GOBIN=$(CURDIR)/bin go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
