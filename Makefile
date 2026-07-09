.PHONY: doc-serve doc-build doc-deploy dev-up dev-down dev-build dev-restart

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