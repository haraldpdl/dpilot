# Quality gates. `make ci` is exactly what GitHub Actions runs; `make hooks`
# wires the same gates into this clone (pre-commit: quick, pre-push: ci).
GO ?= go
GOFMT := $(shell $(GO) env GOROOT)/bin/gofmt
STATICCHECK := honnef.co/go/tools/cmd/staticcheck@2026.2.1
GOVULNCHECK := golang.org/x/vuln/cmd/govulncheck@v1.8.0
TESTFLAGS ?=

.PHONY: ci quick fmt fmt-check vet lint tidy-check test build cross vuln integration hooks

ci: fmt-check vet lint tidy-check test build cross vuln

quick: fmt-check vet

fmt:
	$(GOFMT) -w .

fmt-check:
	@set -e; out="$$($(GOFMT) -l .)"; if [ -n "$$out" ]; then echo "gofmt needed on:"; echo "$$out"; exit 1; fi

vet:
	$(GO) vet ./...

lint:
	$(GO) run $(STATICCHECK) ./...

tidy-check:
	$(GO) mod tidy -diff

test:
	$(GO) test $(TESTFLAGS) ./...

build:
	$(GO) build ./...

# Every goreleaser target must at least compile, with CGO off as in the release.
cross:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -o /dev/null ./cmd/dpilot
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GO) build -o /dev/null ./cmd/dpilot
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GO) build -o /dev/null ./cmd/dpilot
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GO) build -o /dev/null ./cmd/dpilot

vuln:
	$(GO) run $(GOVULNCHECK) ./...

# Needs a real ddev with a project named dpilot-fixture; not part of ci.
integration:
	$(GO) test -tags integration ./integration/...

hooks:
	git config core.hooksPath .githooks
	@echo "hooks installed: pre-commit runs 'make quick', pre-push runs 'make ci'"
