.PHONY: doc-serve doc-build doc-deploy \
	dev-up dev-down dev-build dev-restart \
	build test test-web test-all fmt tidy verify \
	run-controlplane run-gateway run-worker run-all \
	seed snapshot-publish \
	deploy-init deploy-up deploy-down deploy-logs deploy-health \
	build-release build-images brand-assets

GO ?= go
BIN_DIR ?= bin
DEPLOY_COMPOSE ?= deploy/docker-compose.yml
DEPLOY_ENV ?= deploy/.env

brand-assets:
	./scripts/generate-brand-assets.sh

doc-serve:
	uvx --from mkdocs-material mkdocs serve

doc-build:
	uvx --from mkdocs-material mkdocs build

doc-deploy:
	uvx --from mkdocs-material mkdocs gh-deploy

dev-up:
	docker compose -f docker-compose.yml up -d

dev-down:
	docker compose -f docker-compose.yml down

dev-build:
	docker compose -f docker-compose.yml build

dev-restart:
	docker compose -f docker-compose.yml restart

fmt:
	$(GO) fmt ./...

tidy:
	$(GO) mod tidy

build:
	mkdir -p $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/controlplane ./cmd/controlplane
	$(GO) build -o $(BIN_DIR)/gateway ./cmd/gateway
	$(GO) build -o $(BIN_DIR)/worker ./cmd/worker
	$(GO) build -o $(BIN_DIR)/afi ./cmd/cli

build-release:
	bash scripts/build-release.sh

build-images:
	@test -f $(DEPLOY_ENV) || (echo "missing $(DEPLOY_ENV) — run make deploy-init" >&2; exit 1)
	docker compose -f $(DEPLOY_COMPOSE) --env-file $(DEPLOY_ENV) build

test:
	$(GO) test ./...

test-web:
	cd web && pnpm test

# Go unit/integration tests plus web vitest suite.
test-all: test test-web

verify:
	bash scripts/verify-local.sh

run-controlplane:
	$(GO) run ./cmd/controlplane

run-gateway:
	$(GO) run ./cmd/gateway

run-worker:
	$(GO) run ./cmd/worker

# Background CP + worker, foreground gateway.
run-all:
	@echo "Starting control plane on :8081..."
	@$(GO) run ./cmd/controlplane & echo $$! > .controlplane.pid
	@sleep 1
	@echo "Starting worker..."
	@$(GO) run ./cmd/worker & echo $$! > .worker.pid
	@sleep 1
	@echo "Starting gateway on :8080 (Ctrl+C stops gateway; make stop-all cleans up)..."
	@$(GO) run ./cmd/gateway; \
		ec=$$?; \
		$(MAKE) stop-all; \
		exit $$ec

stop-all:
	@if [ -f .controlplane.pid ]; then kill $$(cat .controlplane.pid) 2>/dev/null || true; rm -f .controlplane.pid; fi
	@if [ -f .worker.pid ]; then kill $$(cat .worker.pid) 2>/dev/null || true; rm -f .worker.pid; fi
	@pkill -f 'go run ./cmd/controlplane' 2>/dev/null || true
	@pkill -f 'go run ./cmd/gateway' 2>/dev/null || true
	@pkill -f 'go run ./cmd/worker' 2>/dev/null || true

seed:
	$(GO) run ./cmd/cli seed

snapshot-publish:
	$(GO) run ./cmd/cli snapshot publish

# --- Self-hosted deploy (Docker Compose) ---
# See docs/deployment.md and docs/deployment/customization.md

deploy-init:
	@test -f deploy/.env || cp deploy/env.example deploy/.env
	@test -f deploy/afi.yaml || cp deploy/afi.example.yaml deploy/afi.yaml
	@echo "Wrote deploy/.env and/or deploy/afi.yaml if missing."
	@echo "Replace every CHANGE_ME value, then run: make deploy-up"

deploy-up:
	bash scripts/deploy-up.sh

deploy-down:
	bash scripts/deploy-down.sh

deploy-logs:
	@test -f $(DEPLOY_ENV) || (echo "missing $(DEPLOY_ENV) — run make deploy-init" >&2; exit 1)
	docker compose -f $(DEPLOY_COMPOSE) --env-file $(DEPLOY_ENV) logs -f

deploy-health:
	bash scripts/deploy-health.sh
