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

# apple-schemas regenerates both embedded Apple schema tables from apple/device-management:
# internal/common/appleprofiles/profiles.json (configuration profile payloads) and
# internal/common/appledeclarations/declarations.json (declarative device management declarations,
# plus the status-item vocabulary that status-subscriptions accepts). One clone, one generator and
# one commit per branch, so the two tables can never be built from different upstream revisions —
# a legacy payload and the declaration wrapping it are validated against the same snapshot.
#
# It is deliberately NOT part of `make generate`: it needs network access and a clone. Review the
# diff and commit the regenerated tables.
#
# Both tables are the UNION of the release branch and Apple's newest seed (pre-release) branch,
# because Jamf's generative-declarations service tracks seed: as of 2026-09 seed_OS_27_0 carries 12
# configuration declaration types and keys such as siri.settings.AllowSiriAI that release does not,
# all of them already offered in the Jamf UI. A table built from release alone would report those as
# unknown — and an unknown name is an error, not a warning, so that would block working configs.
#
# The seed branch is discovered rather than pinned, since its name moves with the OS
# (seed_OS_27_0 -> seed_OS_28_0). If discovery finds nothing the target FAILS instead of quietly
# building from release alone, which would turn every seed-only key into an error. Override to pin:
#   make apple-schemas SEED_REF=seed_OS_27_0
RELEASE_REF ?= release
SEED_REF ?=
apple-schemas:
	@set -e; \
	seed='$(SEED_REF)'; \
	if [ -z "$$seed" ]; then \
		echo "Discovering Apple's newest seed branch..."; \
		seed="$$(git ls-remote --heads https://github.com/apple/device-management.git 'seed_OS_*' \
			| sed 's#.*refs/heads/##' | sort -V | tail -1)"; \
	fi; \
	if [ -z "$$seed" ]; then \
		echo "apple-schemas: no seed_OS_* branch found upstream." >&2; \
		echo "  Apple may have changed the naming convention. Building from $(RELEASE_REF) alone" >&2; \
		echo "  would report every seed-only key as unknown, which is now an error, so refusing." >&2; \
		echo "  Investigate, then pin explicitly: make apple-schemas SEED_REF=<branch>" >&2; \
		exit 1; \
	fi; \
	echo "Using seed branch $$seed"; \
	work="$$(mktemp -d)"; \
	trap 'rm -rf "$$work"' EXIT; \
	for ref in '$(RELEASE_REF)' "$$seed"; do \
		echo "Cloning apple/device-management ($$ref)..."; \
		git clone --depth 1 --branch "$$ref" --filter=blob:none --sparse \
			https://github.com/apple/device-management.git "$$work/$$ref" >/dev/null 2>&1; \
		git -C "$$work/$$ref" sparse-checkout set mdm/profiles declarative >/dev/null 2>&1; \
	done; \
	cd tools && go run ./appleprofiles \
		-root '$(RELEASE_REF)'="$$work/$(RELEASE_REF)" \
		-commit '$(RELEASE_REF)'="$$(git -C "$$work/$(RELEASE_REF)" rev-parse HEAD)" \
		-root "$$seed=$$work/$$seed" \
		-commit "$$seed=$$(git -C "$$work/$$seed" rev-parse HEAD)" \
		-release "$$(git -C "$$work/$$seed" log -1 --format=%s)" \
		-profiles-out ../internal/common/appleprofiles/profiles.json \
		-declarations-out ../internal/common/appledeclarations/declarations.json

# apple-profiles is the previous name of apple-schemas, kept so existing muscle memory and any
# external reference keep working.
apple-profiles: apple-schemas

# permissions-map refreshes internal/common/permissions/permissions-map.md, the committed markdown
# rendering of Jamf's "Jamf Pro permissions map" article. TestCatalogueMatchesThePublishedMap
# asserts every row of catalogue.go against it, so the snapshot is what makes a renamed or moved
# permission a build failure instead of a silently wrong docs table.
#
# Like apple-profiles this is deliberately NOT part of `make generate`: it needs network access, and
# the article is revised occasionally and unpredictably. Fetches to a temp file first so a dropped
# connection can never truncate the tracked snapshot in place. Review the diff, then `make test`.
permissions-map:
	@curl -fsS --remove-on-error \
		https://developer.jamf.com/platform-api/reference/jamf-pro-permissions-map.md \
		-o internal/common/permissions/permissions-map.md.tmp
	@mv internal/common/permissions/permissions-map.md.tmp internal/common/permissions/permissions-map.md
	@echo "refreshed internal/common/permissions/permissions-map.md — review the diff, then: make test"

# fix rewrites deprecated API usages. It runs each of the three build contexts
# .github/workflows/integration-tests.yml gates on, because a rewrite that lands
# only in a file behind a build tag is invisible to the untagged pass and used to
# reach CI as a drift failure. Repeat until clean: go fix is not idempotent in one
# pass, since a pass that stamps `//go:fix inline` on a helper only enables the
# NEXT pass to inline its call sites.
#
# Which rewrites it proposes depends on the toolchain, so run it under the version
# `go.mod` names — a newer local Go can drop a modernizer CI still applies, and
# then agree that a tree CI rejects is already at a fixpoint.
# A `toolchain` directive wins over the `go` line, matching how the go command
# itself resolves the version.
GO_MOD_TOOLCHAIN := $(shell awk '/^toolchain /{print $$2; found=1} /^go /{if (!found) v="go" $$2} END{if (!found) print v}' go.mod)

fix: export GOTOOLCHAIN = $(GO_MOD_TOOLCHAIN)
fix:
	go fix ./...
	go fix -tags acceptance ./...
	go fix -tags acctargets,acclanes ./scripts/...

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

.PHONY: fmt fix lint apple-schemas apple-profiles permissions-map test test-scripts testacc testacc-run testacc-changed acclanes-preview build install generate