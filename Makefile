.DEFAULT_GOAL := help

.PHONY: help setup fmt lint test compose-validate up down smoke integration load backup alert-cycle kind-up kind-down

help: ## List available targets.
	@awk 'BEGIN {FS = ":.*## "; printf "Available targets:\n"} /^[a-zA-Z_-]+:.*?## / {printf "  %-18s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

setup: ## Create the local .env file when it does not exist.
	./scripts/setup.sh

fmt: ## Format Go code.
	gofmt -w cmd internal migrations

lint: ## Run static checks included with Go.
	go vet ./...

test: ## Run unit tests with the race detector and coverage.
	go test -race -coverprofile=coverage.out ./...

compose-validate: ## Render and validate the Compose model.
	docker compose config --quiet

up: setup ## Build and start the full local platform.
	docker compose up --build -d --wait

down: ## Stop the platform while preserving named volumes.
	docker compose down

smoke: ## Check every local service and observability endpoint.
	./scripts/smoke-test.sh

integration: ## Execute successful, idempotent and failed import scenarios.
	./scripts/integration-test.sh

load: ## Execute the k6 load test and save reports under artifacts/.
	./scripts/load-test.sh

alert-cycle: ## Print the last firing/resolved cycle recorded by the alert recorder.
	sh scripts/alert-cycle.sh $(ALERT) $(INSTANCE)

backup: ## Dump, restore and compare the PostgreSQL database safely.
	./scripts/verify-backup.sh

kind-up: ## Create the local Kubernetes lab.
	./scripts/kind-up.sh

kind-down: ## Delete only the named local Kubernetes lab cluster.
	./scripts/kind-down.sh
