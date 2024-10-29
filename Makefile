FIX=1
COMMON_TEST_OPTIONS=-race
CMDS=cmd/github-next-semantic-version/github-next-semantic-version
BUILDARGS=

default: help

.PHONY: build
build: $(CMDS) ## Build Go binaries

cmd/github-next-semantic-version/github-next-semantic-version: $(shell find cmd/github-next-semantic-version internal -type f -name '*.go')
	cd `dirname $@` && export CGO_ENABLED=0 && go build $(BUILDARGS) -o `basename $@` *.go

.PHONY: gofmt
gofmt:
	@if test "$(FIX)" = "1"; then \
		set -x ; gofmt -s -w . ;\
	else \
		set -x ; gofmt -s -d . ;\
	fi

.PHONY: golangcilint
golangcilint: tmp/bin/golangci-lint
	@if test "$(FIX)" = "1"; then \
		set -x ; $< run --fix --timeout 10m;\
	else \
		set -x ; $< run --timeout 10m;\
	fi

.PHONY: govet
govet:
	go vet ./...

.PHONY: test-unit
test-unit: ## Execute all unit tests
	go test $(COMMON_TEST_OPTIONS) ./...

.PHONY: test
test: test-unit test-integration ## Execute all tests 

.PHONY: lint
lint: govet gofmt golangcilint ## Lint the code (also fix the code if FIX=1, default)

tmp/bin/golangci-lint:
	@mkdir -p tmp/bin
	cd tmp/bin && curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b . v1.56.2 && chmod +x `basename $@`

.PHONY: clean
clean: _cmd_clean ## Clean the repo
	rm -Rf tmp
	rm -Rf build

.PHONY: clean
doc: $(CMDS) ## Generate documentation
	docker run -t -v $$(pwd):/workdir --user=$$(id -u) ghcr.io/fabien-marty/jinja-tree:latest /workdir

.PHONY: _cmd_clean
_cmd_clean:
	rm -f $(CMDS)

.PHONY: no-dirty
no-dirty: ## Check if the repo is dirty
	@if test -n "$$(git status --porcelain)"; then \
		echo "ERROR: the repository is dirty"; \
		git status; \
		git diff; \
		exit 1; \
	fi

.PHONY: test-integration
test-integration: $(CMDS) ## Run integration tests
	N=`./cmd/github-next-semantic-version/github-next-semantic-version --log-level=DEBUG .`; \
	LINES=`echo $$N |wc -l`; \
	if test "$${LINES}" != "1"; then \
		echo "Expected 1 line, got $${LINES}"; \
		exit 1; \
	fi; \
	echo "$$N" |grep '^v'
	
.PHONY: help
help::
	@# See https://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
