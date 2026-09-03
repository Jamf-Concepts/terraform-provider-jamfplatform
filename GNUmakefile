default: fmt lint install generate

build:
	go build -v ./...

install: build
	go install -v ./...

# lint covers the default build plus the build-tagged tooling under scripts/,
# which the default invocation cannot see.
lint:
	golangci-lint run
	golangci-lint run --build-tags acctargets ./scripts/acctargets/

generate:
	cd tools; go generate ./...

# fmt scopes gofmt to the Go source trees that ship with the provider so that
# generated fixtures, examples, and local-testing scratch dirs are left alone.
# Add new top-level Go packages here if any are introduced.
fmt:
	gofmt -s -w -e main.go internal/ tools/ scripts/

# apple-profiles regenerates the embedded Apple configuration profile schema table at
# internal/common/appleprofiles/profiles.json from apple/device-management. It is deliberately NOT
# part of `make generate`: it needs network access and a clone, and the upstream schemas only change
# a few times a year, tracking Apple OS releases. Run it when Apple ships a release, review the diff,
# and commit the regenerated table.
#
# Override REF to pin a different upstream branch or tag:
#   make apple-profiles REF=release
REF ?= release
apple-profiles:
	@set -e; \
	work="$$(mktemp -d)"; \
	trap 'rm -rf "$$work"' EXIT; \
	echo "Cloning apple/device-management ($(REF))..."; \
	git clone --depth 1 --branch '$(REF)' --filter=blob:none --sparse \
		https://github.com/apple/device-management.git "$$work/src" >/dev/null 2>&1; \
	git -C "$$work/src" sparse-checkout set mdm/profiles >/dev/null 2>&1; \
	commit="$$(git -C "$$work/src" rev-parse HEAD)"; \
	release="$$(git -C "$$work/src" log -1 --format=%s)"; \
	cd tools && go run ./appleprofiles \
		-source "$$work/src/mdm/profiles" \
		-commit "$$commit" \
		-release "$$release" \
		-ref '$(REF)' \
		-out ../internal/common/appleprofiles/profiles.json

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

# test-scripts runs the unit tests for the build-tagged tooling under scripts/,
# which `go test ./...` cannot see. Each tool needs its own -tags invocation:
# the tag that makes one visible does not make the other's files build.
test-scripts:
	go test -count=1 -tags acctargets ./scripts/acctargets/
	go test -count=1 -tags acclanes ./scripts/acclanes/

# testacc-changed runs acceptance tests only for the packages affected by the
# current change set: the changed packages plus everything that transitively
# depends on them (see scripts/acctargets). Override BASE to diff against a ref
# other than origin/main:
#   make testacc-changed
#   make testacc-changed BASE=origin/feat/pro-expansion
BASE ?=
testacc-changed:
	@pkgs="$$(go run -tags acctargets ./scripts/acctargets $(BASE))"; \
	if [ -z "$$pkgs" ]; then \
		echo "No acceptance packages affected by the current changes."; \
		exit 0; \
	fi; \
	echo "Acceptance scope: $$pkgs"; \
	TF_ACC=1 go test -v -cover -count=1 -tags acceptance -timeout 120m -p=1 $$pkgs

# acclanes-preview prints the GitHub Actions matrix the acceptance workflow would
# build for the current change set, so an edit to .github/acceptance-lanes.json
# can be checked here rather than by pushing and reading a plan job. Companion to
# testacc-changed: that target says WHICH packages run, this one says which LANE
# each lands in and on whose credentials. Override BASE the same way, or pass
# SCOPE=./... to preview the full suite:
#   make acclanes-preview
#   make acclanes-preview SCOPE=./...
SCOPE ?=
acclanes-preview:
	@scope="$(SCOPE)"; \
	if [ -z "$$scope" ]; then \
		scope="$$(go run -tags acctargets ./scripts/acctargets $(BASE))"; \
	fi; \
	if [ -z "$$scope" ]; then \
		echo "No acceptance packages affected by the current changes; the matrix would be []."; \
		exit 0; \
	fi; \
	go run -tags acclanes ./scripts/acclanes -scope "$$scope"

.PHONY: fmt fix lint apple-profiles test test-scripts testacc testacc-run testacc-changed acclanes-preview build install generate