default: fmt lint install generate

build:
	go build -v ./...

install: build
	go install -v ./...

lint:
	golangci-lint run

generate:
	cd tools; go generate ./...

# fmt scopes gofmt to the Go source trees that ship with the provider so that
# generated fixtures, examples, and local-testing scratch dirs are left alone.
# Add new top-level Go packages here if any are introduced.
fmt:
	gofmt -s -w -e main.go internal/ tools/ scripts/

fix:
	go fix ./...

test:
	go test -v -cover -count=1 -timeout=120s -p=10 ./...

# testacc runs every acceptance test in the repository serially against a real
# Jamf Platform tenant. Requires JAMFPLATFORM_BASE_URL / CLIENT_ID / CLIENT_SECRET
# / TENANT_ID in the environment.
testacc:
	TF_ACC=1 go test -v -cover -count=1 -tags acceptance -timeout 120m -p=1 ./...

# testacc-run targets a subset of acceptance tests. Override RUN (Go -run regex)
# and PKG (package path) on the command line. Defaults: every test in every package,
# i.e. the same scope as `make testacc` but accepts TESTARGS for extra flags.
#
# Examples:
#   make testacc-run RUN=TestAccResource_ProNetworkSegment_Basic \
#     PKG=./internal/resources/pro/network_segment/...
#   make testacc-run TESTARGS='-failfast'
RUN ?= .
PKG ?= ./...
TESTARGS ?=
testacc-run:
	TF_ACC=1 go test -v -cover -count=1 -tags acceptance -timeout 120m -p=1 -run '$(RUN)' $(TESTARGS) $(PKG)

# testacc-changed runs acceptance tests only for the packages affected by the
# current change set: the changed packages plus everything that transitively
# depends on them (see scripts/acctargets). Override BASE to diff against a ref
# other than origin/main:
#   make testacc-changed
#   make testacc-changed BASE=origin/feat/pro-expansion
BASE ?=
testacc-changed:
	@pkgs="$$(go run scripts/acctargets/main.go $(BASE))"; \
	if [ -z "$$pkgs" ]; then \
		echo "No acceptance packages affected by the current changes."; \
		exit 0; \
	fi; \
	echo "Acceptance scope: $$pkgs"; \
	TF_ACC=1 go test -v -cover -count=1 -tags acceptance -timeout 120m -p=1 $$pkgs

.PHONY: fmt fix lint test testacc testacc-run testacc-changed build install generate