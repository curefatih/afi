.PHONY: serve build deploy

serve:
	uvx --from mkdocs-material mkdocs serve

build:
	uvx --from mkdocs-material mkdocs build

deploy:
	uvx --from mkdocs-material mkdocs gh-deploy