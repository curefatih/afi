.PHONY: serve build deploy

doc-serve:
	uvx --from mkdocs-material mkdocs serve

doc-build:
	uvx --from mkdocs-material mkdocs build

doc-deploy:
	uvx --from mkdocs-material mkdocs gh-deploy