default: help

github-next-semantic-version: ../cmd/github-next-semantic-version/github-next-semantic-version
	cp -f $< .

.PHONY: build-prepare
build-prepare: github-next-semantic-version ## Prepare build

.PHONY: build
build: build-prepare ## Build docker image
	docker build -t gnsv:latest .

clean: ## Clean up
	rm -f github-next-semantic-version

.PHONY: help
help::
	@# See https://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
