# Repository Guidelines

## Overview

Terraform provider for Jamf Platform APIs, built on the [Terraform Plugin Framework](https://github.com/hashicorp/terraform-plugin-framework) v1.19.0 with Protocol v6. Go module path: `github.com/Jamf-Concepts/terraform-provider-jamfplatform`.

The provider exposes four Terraform construct types: **resources** (CRUD-managed objects), **data sources** (read-only lookups), **list resources** (query-based streaming with RSQL filters), and **actions** (fire-and-forget device management commands).

The Jamf Platform API client is an **external** Go SDK consumed as a dependency: `github.com/Jamf-Concepts/jamfplatform-go-sdk` (package `jamfplatform`). It is not vendored under `internal/`.

## Companion Docs

CLAUDE.md is a quick orientation. The following root-level docs are authoritative on conflict:

- `STYLE_GUIDE.md` — canonical Go style, file-convention table, dependency policy, Jamf Pro naming/version/endpoint policies, ID/import conventions.
- `TESTING.md` — test categories, commands, build tags, CI scaling plan.
- `CONTRIBUTING.md` — contribution workflow, release-versioning policy.
- `README.md` — user-facing provider usage.

## Project Structure

```
internal/
├── provider/          # Provider config, registration, and logging
├── resources/         # Resource, data source, and list resource implementations
│   ├── blueprints/    # blueprint/, blueprints/, component/, components/    (Platform Services)
│   ├── cbengine/      # benchmark/, benchmarks/, baselines/, rules/         (Platform Services)
│   ├── device/        # Single device data source                            (Platform Services)
│   ├── device_group/  # Resource, data source, list resource                 (Platform Services)
│   ├── device_groups/ # Plural data source                                   (Platform Services)
│   ├── devices/       # Plural data source                                   (Platform Services)
│   └── pro/           # Jamf Pro resources — two-tier domain grouping (planned)
│       ├── computers/         # computer, computer_group, smart_computer_group, ...
│       ├── mobile_devices/    # mobile_device, mobile_device_group, ...
│       ├── users/             # user, user_group, ...
│       ├── policies/          # policy, script, package, restricted_software, ...
│       ├── configuration_profiles/  # macos_configuration_profile, mobile_device_configuration_profile
│       ├── enrollment/        # prestages, onboarding, enrollment_customization, ...
│       ├── sso/               # sso_settings, sso_certificate, oidc, ldap, ...
│       ├── patch/             # patch_policy, patch_software_title_configuration, ...
│       ├── vpp/               # volume_purchasing_*, vpp_*
│       ├── inventory/         # site, building, category, department, network_segment, ...
│       └── settings/          # singletons: activation_code, client_check_in, jamf_pro_*, ...
├── actions/
│   └── device/        # erase, restart, shutdown, unmanage
├── common/
│   ├── helpers/       # Type conversions, polling, timeout, state reconciliation, dynamic JSON
│   └── filters/       # RSQL filter schema blocks and expression builder
└── testhelpers/       # Acceptance test client/provider factories and mock server
tools/                 # go:generate entrypoint (copywrite, terraform fmt, tfplugindocs)
testing/               # Terraform-native integration tests (.tftest.hcl, .tfquery.hcl)
local-testing/         # Manual API request workflows for development
examples/
├── provider/          # Example provider block
├── resources/         # Example .tf for resources
├── data-sources/      # Example .tf for data sources
├── list-resources/    # Example .tf for list resources
└── actions/           # Example .tf for actions
docs/                  # Auto-generated provider documentation (do not hand-edit)
GNUmakefile            # build / install / lint / generate / fmt / test / testacc targets
goreleaser.yml         # Release build config
.copywrite.hcl         # Copyright header config (used by tools/tools.go)
```

Resources grouped by API domain. Within a domain, primary resource package (e.g. `blueprint/`) contains resource, data source, list resource. Sibling packages (e.g. `blueprints/`, `components/`) hold related plural or secondary data sources. Complex resources may use sub-packages for nested payload builders (e.g. `blueprints/blueprint/components/`).

**Reference implementation for complex resources**: `internal/resources/blueprints/blueprint/`. Demonstrates the full file split, schema versioning with `state_upgrader.go`, and nested payload builders via a `components/` sub-package. Mirror its layout when adding non-trivial CRUD resources.

**Jamf Pro resources (planned rollout)**: live under `internal/resources/pro/<domain>/<resource>/`. Terraform construct name format: `jamfplatform_pro_<resource>` regardless of whether the SDK source is `pro/` or `proclassic/` — users do not need to know the API-layer split. Credentials are the same `JAMFPLATFORM_*` set used by Platform Services resources (one client serves everything). Each Pro resource declares an unexported `const minJamfProVersion` and funnels its Configure through `providerdata.ConfigurePro` — the shared helper that performs the type assertion, fetches the tenant version via `sync.Once`-cached `GetJamfProVersionV1`, runs the per-resource gate when set, and appends the provider-floor advisory warning. Do not hand-roll Configure boilerplate. Reference template: `internal/resources/pro/inventory/category/`. Platform Services resources are unaffected — the Pro version is fetched only when a Pro construct with a non-empty `minJamfProVersion` is in the config. If the tenant lacks Jamf Pro, the version fetch errors and surfaces at Configure time. Endpoint versions are tracked via an annotation block at the top of each Pro resource's `crud.go` (`SDK endpoints used: ... Status: current. Last reviewed YYYY-MM-DD.`) with a 6-month soft / 3-month hard migration policy on deprecation. This policy applies **only** to Pro / ProClassic SDK packages — Platform Services SDK packages (`blueprints/`, `compliancebenchmarks/`, `devicegroups/`, `devices/`, `deviceactions/`, `ddmreport/`) are continuously-deployed microservices and resources backed by them stay current with no annotation or buffer. See [STYLE_GUIDE.md §Jamf Pro Resource Naming](STYLE_GUIDE.md#jamf-pro-resource-naming), [§Minimum Jamf Pro version check](STYLE_GUIDE.md#minimum-jamf-pro-version-check), and [§Endpoint adoption & migration policy](STYLE_GUIDE.md#endpoint-adoption--migration-policy) for full rules; [CONTRIBUTING.md §Adding a Jamf Pro Resource](CONTRIBUTING.md#adding-a-jamf-pro-resource) for the SDK-comparison + maintainer-approval workflow. Local planning inventory: `JAMF_PRO_INVENTORY.md` (gitignored).

## Tooling

- Go >= 1.26, Terraform >= 1.13.0.
- `GNUmakefile` is the canonical entrypoint. Default target: `fmt lint install generate`.
- Releases built with goreleaser (`goreleaser.yml`).
- Doc + header generation runs through `tools/tools.go` via `make generate` (which `cd tools && go generate ./...`): copyright headers (`hashicorp/copywrite`), `terraform fmt -recursive ../examples/`, and provider docs (`hashicorp/terraform-plugin-docs/cmd/tfplugindocs`).

### Available make targets

| Target     | Description                                                |
| ---------- | ---------------------------------------------------------- |
| `build`    | Build the provider                                         |
| `install`  | Build and install the provider locally                     |
| `fmt`      | Format Go source files (`gofmt -s -w -e .`)                |
| `fix`      | Run `go fix ./...` — rewrites deprecated API usages         |
| `lint`     | Run `golangci-lint`                                        |
| `generate` | Copyright headers + `terraform fmt examples/` + docs       |
| `test`     | Run unit tests (excludes `acceptance` build tag)           |
| `testacc`  | Run acceptance tests (sets `TF_ACC=1`, requires tenant)    |

## Jamf Platform API

- RESTful, multiple endpoints. Auth and HTTP flows live in the external `jamfplatform-go-sdk` module — not in this repo.
- SDK uses explicit version suffixes on types/functions (e.g. `CreateBlueprintV1`, `CreateCBEngineBenchmarkV2`) for multi-version support. See `STYLE_GUIDE.md` §Client Conventions for the naming pattern.
- API spec available piecemeal on request.

## Provider Development

- All Terraform Plugin Framework code lives in `internal/`.
- SDK import: `github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform`. Bump `go.mod` to pick up new endpoints.
- Shared utilities: `internal/common/helpers` (type conversions, polling, timeout, state reconciliation, error detection, dynamic JSON) and `internal/common/filters` (RSQL filter blocks + expression builder).
- Acceptance test fixtures (provider factories, real-client builder, mock HTTP server) live in `internal/testhelpers`.
- Before committing: `make fix fmt lint test`. Regenerate docs: `make generate`.
- Every Go file carries `// Copyright Jamf Software LLC <year>` + `// SPDX-License-Identifier: MPL-2.0` headers (managed by `copywrite` via `make generate`). 2026

## Style, Conventions, Schema

Authoritative source: **`STYLE_GUIDE.md`**. Covers Go conventions, dependency policy, resource package file conventions (`resource.go`, `crud.go`, `model_types.go`, `schema_types.go`, `mappings.go`, `input_builders.go`, `state_builders.go`, `helpers.go`, `state_upgrader.go`, `plan_modifiers.go`, `validators.go`, `list_resource.go`, `data_source.go`), optional split-outs (`endpoints_*`, `nested_*`), test file conventions, client versioning, schema rules (inline + flat, nested attributes over blocks), Sets vs Lists policy, error handling helpers (`helpers.IsNotFoundError`, `helpers.PollUntil`), and naming patterns.

Mirror the reference implementation `internal/resources/blueprints/blueprint/` when adding a new complex resource.

## Testing

Authoritative source: **`TESTING.md`**. Key commands:

- Unit: `make test` (no API needed; excludes `acceptance` build tag).
- Acceptance: `make testacc` (sets `TF_ACC=1`, requires tenant + credentials, gated by `//go:build acceptance`).

Every acceptance test file **must** declare `//go:build acceptance` on line 1 or it leaks into unit runs. Use factories from `internal/testhelpers` (`AccPreCheck`, `AccTestProtoV6ProviderFactories`).

## Environment Variables

- `JAMFPLATFORM_BASE_URL` — Base URL of the Jamf Platform tenant. Region-specific: `https://us.apigw.jamf.com` (US), `https://eu.apigw.jamf.com` (EU), `https://apac.apigw.jamf.com` (APAC).
- `JAMFPLATFORM_CLIENT_ID` — API client ID for authentication.
- `JAMFPLATFORM_CLIENT_SECRET` — API client secret for authentication.
- Can also be set in the provider block in Terraform configuration.
- Acceptance tests additionally require `TF_ACC=1` (set automatically by `make testacc`).

## Adding a New Resource

1. Ingest the provided OpenAPI spec or request/response examples for the endpoint.
2. Confirm the required SDK functions/types exist in `jamfplatform-go-sdk`. If not, bump the SDK dep or coordinate adding them upstream first.
3. Create the package at `internal/resources/<domain>/<resource>/` with the file split from "Resource Package File Conventions" above (`resource.go`, `crud.go`, `model_types.go`, `schema_types.go`, `mappings.go`, `input_builders.go`, `state_builders.go`, `helpers.go`, plus `list_resource.go` / `data_source.go` as needed).
4. Register the resource, data source, list resource, or action in `internal/provider/provider.go` (`Resources()`, `DataSources()`, `ListResources()`, `Actions()`).
5. Add unit tests in the same package: `schema_test.go`, `input_builders_test.go`, `state_builders_test.go`, plus any helpers/upgrader tests.
6. Add `resource_acceptance_test.go` with `//go:build acceptance` and use factories from `internal/testhelpers`.
7. Add example `.tf` files under the matching `examples/` subdirectory: `examples/resources/<name>/`, `examples/data-sources/<name>/`, `examples/list-resources/<name>/`, or `examples/actions/<name>/`.
8. Run `make fix fmt lint test` and confirm clean (zero lint issues, all unit tests pass). Run `fix` first — it rewrites deprecated API usages so `fmt` and `lint` operate on the migrated source.
9. Run `make generate` to regenerate copyright headers, format examples, and rebuild `docs/`. **Mandatory** for every new resource, data source, list resource, or action — `docs/<construct-type>/<name>.md` must land in the same PR.

## Documentation & Examples

- Update `examples/` when adding or changing resources, data sources, list resources, or actions.
- `docs/` is generated — never hand-edit. Regenerate with `make generate`.
- Schema `Description` / `MarkdownDescription` fields are the source of truth for generated docs.
