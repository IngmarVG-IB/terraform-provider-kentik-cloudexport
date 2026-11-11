HOSTNAME  = registry.terraform.io
NAMESPACE = kentik
NAME      = kentik-cloudexport
BINARY    = terraform-provider-${NAME}
VERSION  ?= $(shell git describe --tags --abbrev=0 2>/dev/null | sed -e 's/^v//' -e 's/^$$/0.0.0-dev/')
OS_ARCH  := $(shell go env GOHOSTOS)_$(shell go env GOHOSTARCH)

# Pinned tool versions. Run via `go run` so they are built with the local Go toolchain.
GOLANGCI_LINT_VERSION ?= v2.15.0
GOVULNCHECK_VERSION   ?= v1.9.0
GOLANGCI_LINT ?= go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
GOVULNCHECK   ?= go run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)

default: build

.PHONY: build
build:
	go build -trimpath -o ${BINARY} .

.PHONY: install
install: build
	mkdir -p ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${VERSION}/${OS_ARCH}
	mv ${BINARY} ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${VERSION}/${OS_ARCH}

# Offline unit tests: fake gRPC server, no Kentik credentials needed.
# The terraform-plugin-testing harness needs a terraform (or tofu) binary on PATH
# or in TF_ACC_TERRAFORM_PATH.
.PHONY: test
test:
	go test ./... -count=1 -timeout=10m

# Acceptance tests against a live Kentik tenant. Requires TF_ACC=1,
# KTAPI_AUTH_EMAIL, KTAPI_AUTH_TOKEN and KENTIK_PLAN_ID. Never run in default CI.
.PHONY: testacc
testacc:
	@test -n "$$KTAPI_AUTH_EMAIL" -a -n "$$KTAPI_AUTH_TOKEN" || (echo "KTAPI_AUTH_EMAIL and KTAPI_AUTH_TOKEN must be set" >&2; exit 1)
	TF_ACC=1 go test ./... -count=1 -v -timeout=30m

.PHONY: lint
lint:
	$(GOLANGCI_LINT) run --allow-parallel-runners ./...

.PHONY: fmt
fmt:
	$(GOLANGCI_LINT) fmt ./...

.PHONY: vuln
vuln:
	$(GOVULNCHECK) ./...

# Regenerates everything that is checked in: currently the registry docs.
.PHONY: generate
generate: docs

.PHONY: docs
docs:
	go tool tfplugindocs generate --provider-name ${NAME}
	go tool tfplugindocs validate --provider-name ${NAME}

# Fails if `make generate` changes tracked files (used by CI).
.PHONY: check-generate
check-generate: generate
	git diff --exit-code -- docs
	@test -z "$$(git status --porcelain -- docs)" || (git status --porcelain -- docs; echo "docs are stale, run 'make generate'" >&2; exit 1)

.PHONY: check-go-mod
check-go-mod:
	go mod tidy -diff

# Validates the kentik-api.json coverage manifest.
.PHONY: check-api-manifest
check-api-manifest:
	./tools/check_api_manifest.sh

.PHONY: clean
clean:
	rm -f ${BINARY}

# Runs `terraform validate` on every example with a dev_overrides CLI config
# (pass TF_CLI=tofu to use OpenTofu). Never contacts the Kentik API.
.PHONY: validate-examples
validate-examples:
	./tools/validate_examples.sh $(TF_CLI)
