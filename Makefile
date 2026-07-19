.PHONY: doc-serve doc-build doc-deploy \
	dev-up dev-down dev-build dev-restart \
	build test fmt tidy verify \
	run-controlplane run-gateway run-all \
	seed snapshot-publish

GO ?= go
BIN_DIR ?= bin

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
	$(GO) build -o $(BIN_DIR)/afi ./cmd/cli

test:
	$(GO) test ./...

verify:
	bash scripts/verify-local.sh

run-controlplane:
	$(GO) run ./cmd/controlplane

run-gateway:
	$(GO) run ./cmd/gateway

# Runs control plane then gateway in the foreground sequentially is not useful;
# use two terminals, or: make run-all (background CP + foreground gateway).
run-all:
	@echo "Starting control plane on :8081..."
	@$(GO) run ./cmd/controlplane & echo $$! > .controlplane.pid
	@sleep 1
	@echo "Starting gateway on :8080 (Ctrl+C stops gateway; kill CP with make stop-all)..."
	@$(GO) run ./cmd/gateway; \
		status=$$?; \
		if [ -f .controlplane.pid ]; then kill $$(cat .controlplane.pid) 2>/dev/null || true; rm -f .controlplane.pid; fi; \
		exit $$status

stop-all:
	@if [ -f .controlplane.pid ]; then kill $$(cat .controlplane.pid) 2>/dev/null || true; rm -f .controlplane.pid; fi
	@pkill -f 'go run ./cmd/controlplane' 2>/dev/null || true
	@pkill -f 'go run ./cmd/gateway' 2>/dev/null || true

seed:
	$(GO) run ./cmd/cli seed

snapshot-publish:
	$(GO) run ./cmd/cli snapshot publish
