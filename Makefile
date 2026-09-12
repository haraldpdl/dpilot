# Quality gates. `make ci` is exactly what GitHub Actions runs; `make hooks`
# wires the same gates into this clone (pre-commit: quick, pre-push: ci).
GO ?= go
STATICCHECK := honnef.co/go/tools/cmd/staticcheck@2026.1
GOVULNCHECK := golang.org/x/vuln/cmd/govulncheck@v1.8.0
TESTFLAGS ?=

.PHONY: ci quick fmt fmt-check vet lint tidy-check test build cross vuln integration hooks

ci: fmt-check vet lint tidy-check test build cross vuln

quick: fmt-check vet

fmt:
	gofmt -w .

fmt-check:
	@out="$$(gofmt -l .)"; if [ -n "$$out" ]; then echo "gofmt needed on:"; echo "$$out"; exit 1; fi

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

# Every goreleaser target must at least compile.
cross:
	GOOS=linux GOARCH=arm64 $(GO) build -o /dev/null ./cmd/dpilot
	GOOS=darwin GOARCH=amd64 $(GO) build -o /dev/null ./cmd/dpilot
	GOOS=darwin GOARCH=arm64 $(GO) build -o /dev/null ./cmd/dpilot

vuln:
	$(GO) run $(GOVULNCHECK) ./...

# Needs a real ddev with a project named dpilot-fixture; not part of ci.
integration:
	$(GO) test -tags integration ./integration/...

hooks:
	git config core.hooksPath .githooks
	@echo "hooks installed: pre-commit runs 'make quick', pre-push runs 'make ci'"
