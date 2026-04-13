# Repository Guidelines

## Overview

This is a Terraform provider for Jamf Platform APIs, built using the [Terraform Plugin Framework](https://github.com/hashicorp/terraform-plugin-framework) v1.17.0 with Protocol v6. The Go module path is `github.com/Jamf-Concepts/terraform-provider-jamfplatform`.

The provider exposes four Terraform construct types: **resources** (CRUD-managed objects), **data sources** (read-only lookups), **list resources** (query-based streaming with RSQL filters), and **actions** (fire-and-forget device management commands).

## Current Resource Inventory

Resources (3): `device_group`, `blueprints_blueprint`, `cbengine_benchmark`.

Data Sources (12): single-item lookups and filtered list sources across all resource domains — blueprints (blueprint, blueprints, component, components), cbengine (benchmark, benchmarks, baselines, rules), devices (device, devices, device_group, device_groups).

List Resources (3): `device_group`, `blueprints_blueprint`, `cbengine_benchmark` — streaming query resources with declarative RSQL filter support.

Actions (4): device management commands — `erase`, `restart`, `shutdown`, `unmanage`.

## Project Structure

```
internal/
├── provider/          # Provider config, registration, and logging
├── resources/         # Resource, data source, and list resource implementations
│   ├── blueprints/    # blueprint/, blueprints/, component/, components/
│   ├── cbengine/      # benchmark/, benchmarks/, baselines/, rules/
│   ├── device/        # Single device data source
│   ├── device_group/  # Resource, data source, list resource
│   ├── device_groups/ # Plural data source
│   └── devices/       # Plural data source
├── actions/           # Fire-and-forget device actions
│   └── device/        # erase, restart, shutdown, unmanage
├── client/            # Jamf Platform API client (OAuth, HTTP, domain methods)
└── common/            # Shared packages
    ├── helpers/       # Type conversions, polling, timeout, state reconciliation
    └── filters/       # RSQL filter schema blocks and expression builder
testing/               # Terraform-native integration tests (.tftest.hcl, .tfquery.hcl)
local-testing/         # Manual API request workflows for development
examples/              # Example .tf configs for all construct types
docs/                  # Auto-generated provider documentation
```

Resources are grouped by API domain. Within a domain, the primary resource package (e.g. `blueprint/`) contains the resource, data source, and list resource. Sibling packages (e.g. `blueprints/`, `components/`) hold related plural or secondary data sources.

## Tooling

- Go >= 1.26, Terraform >= 1.13.0.
- Releases are built with goreleaser (`goreleaser.yml`). There is no Makefile.

## Jamf Platform API

- The Jamf Platform API is RESTful and exposes multiple endpoints — client functions are fully operational and the API spec is available piecemeal upon request. You can see client/auth flows in the `internal/client` package.
- The client uses explicit version suffixes on types and functions (e.g. `CreateBlueprintV1`, `CreateCBEngineBenchmarkV2`) to support multiple API versions without breaking changes. When endpoints are upgraded, new V2 types/functions are added alongside V1 and resources migrate at their own pace.

## Provider Development

- Terraform Plugin Framework code lives in `internal/`.
- The client is in `internal/client`.
- Shared utilities are in `internal/common` — `helpers` (type conversions, polling, state reconciliation, error detection) and `filters` (RSQL filter schema blocks and expression building).
- Resource implementations are grouped by package in `internal/resources/<domain>/<resource>` with files split by concern (crud, helpers, resource, types, data source, list).
- Complex resources may use sub-packages for nested payload builders (e.g. `blueprint/components/`).
- Run formatting and linting before committing: `golangci-lint run` and `go fmt ./...`.
- Generate docs with go's native methodology.
- Run tests with go native methodology. Tests should be split into unit and acceptance tests, and we should not run acceptance tests by default as they require a real Jamf tenant.

## Current State vs Conventions

All existing resource packages have been refactored to follow the conventions below. The three CRUD resource packages (`device_group`, `cbengine/benchmark`, `blueprints/blueprint`) use the full file split (`model_types.go`, `schema_types.go`, `mappings.go`, `input_builders.go`, `state_builders.go`, `helpers.go`, `list_resource.go`). Data-source-only packages use `model_types.go` for their model structs. New resources and future changes should maintain these conventions.

## Code Organization Guidelines

- Look for opportunities to create reusable packages (helper/utility functions) instead of duplicating logic in resource packages.
- Keep packages split by concern with focused files (crud, helpers, resource, types, data source).
- Always look for existing helper functions that can be reused before adding new code.
- Only use native Go, golang/x packages, and Terraform Plugin Framework packages. Avoid third-party dependencies.

## Code Style Guidelines

- Follow Go conventions and idiomatic patterns.
- Favor clear and descriptive naming for variables, functions, and types.
- Always ensure constants, functions, variable sets and types have a short comment describing their purpose.
- Do not add comments inside type definitions or function bodies.

### Resource Package File Conventions

Use resource-agnostic filenames and helper names so the same structure can apply to all resources:

- `resource.go`: schema and boilerplate.
- `crud.go`: Create/Read/Update/Delete and import.
- `model_types.go`: Terraform model structs only.
- `schema_types.go`: attr type maps used to build ObjectValue/ListValue state.
- `mappings.go`: lookup tables and name mappings.
- `input_builders.go`: build API inputs from Terraform model data.
- `state_builders.go`: map API responses to Terraform state.
- `helpers.go`: resource-specific helper functions that don't fit elsewhere.
- `plan_modifiers.go`: schema plan modifiers (if needed).
- `validators.go`: schema validators (if needed).
- `list_resource.go`: for list resources implementing `list.ListResource`.
- `data_source.go`: for data sources implementing `datasource.DataSource`.
- `resource_test.go`: acceptance tests for the resource.

For list resources, follow the framework list resource pattern.

Optional split-outs for complex resources:

- `endpoints_builders.go` and `endpoints_state.go` when endpoint logic dominates.
- `nested_builders.go` and `nested_state.go` for large nested payloads.

## Schema Guidelines

- Schemas should be inline and as flat as possible.
- Favor nested attributes (set/single/list) instead of blocks wherever possible.
- **Sets vs Lists**: Use sets for user-supplied unordered collections where deduplication and order-independent comparison matter (e.g. `members`, `device_groups` scope, `criteria`, `raw_component`, component configuration sets). Use lists for computed API results that are read-only — sets require element hashing which adds CPU overhead with no benefit when the user doesn't control the values. Data source attributes returning API data should always use lists.

## Environment Variables

- `JAMFPLATFORM_BASE_URL` — Base URL of the Jamf Platform tenant (e.g. `https://us.apigw.jamf.com`). Region specific. `https://eu.apigw.jamf.com` for Europe, `https://apac.apigw.jamf.com` for Asia-Pacific.
- `JAMFPLATFORM_CLIENT_ID` — API client ID for authentication.
- `JAMFPLATFORM_CLIENT_SECRET` — API client secret for authentication.
- These can also be set in the provider block in Terraform configuration.

## Testing

- **Unit tests**: `go test ./...` — runs schema validation, metadata, plan modifier, state migration, flattener/expander, helper, and client tests (no real API needed).
- **Acceptance tests**: `go test -tags=acceptance ./... -p=1` — creates real resources against a Jamf Platform tenant. Requires `JAMFPLATFORM_BASE_URL`, `JAMFPLATFORM_CLIENT_ID`, and `JAMFPLATFORM_CLIENT_SECRET` environment variables.
- Test files follow the `*_test.go` convention next to the code they test.

## Adding a New Resource

1. Ask for or ingest the provided OpenAPI specification or request/response examples.
2. Create `internal/provider/<resource_name>_resource.go` implementing `resource.Resource` with CRUD + `ImportState`.
3. Register the resource in `provider.go` → `Resources()`.
4. Create `internal/provider/<resource_name>_resource_test.go` with acceptance tests.
5. Add schema validation tests in `schema_test.go`.
6. Update `examples/` with example `.tf` files.
7. Run `go test ./...` to ensure tests pass.
8. Run the recommended go command to generate documentation from schema descriptions.

## Documentation & Examples

- Update `examples/` when adding new resources or data sources.
- Run the recommended go command to regenerate documentation from schema descriptions.
