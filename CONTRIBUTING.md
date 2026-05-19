# Contributing

Thank you for your interest in contributing to the Terraform Provider for Jamf Platform.

## Prerequisites

- **Go** >= 1.26 (see `go.mod` for the exact version)
- **Terraform** >= 1.13.0 (or **OpenTofu** >= 1.6.0)
- **golangci-lint** for linting
- A Jamf Platform tenant with API credentials (for acceptance tests only)

## Getting Started

```bash
# Clone the repository
git clone https://github.com/Jamf-Concepts/terraform-provider-jamfplatform.git
cd terraform-provider-jamfplatform

# Install dependencies
go mod download

# Build + unit tests
make build
make test
```

## Development Workflow

1. Create a feature branch from `main`.
2. Make your changes following the [style guide](STYLE_GUIDE.md).
3. Run formatting, linting, and tests before committing:

   ```bash
   make fmt
   make lint
   make test
   ```

4. Open a pull request against `main`. CI runs build, lint, docs-generation check, and unit tests automatically.
5. CI also runs the Go acceptance test suite against a real tenant via the GitHub `acceptance` environment after a reviewer approves the run. See [TESTING.md](TESTING.md) for details.

## Adding a New Resource

1. Obtain the OpenAPI specification or request/response examples for the Jamf Platform endpoint.
2. Confirm the required client methods exist in [`jamfplatform-go-sdk`](https://github.com/Jamf-Concepts/jamfplatform-go-sdk). If not, add them upstream (versioned naming, e.g. `CreateMyResourceV1`) and bump the dep in `go.mod`.
3. Create the resource package under `internal/resources/<domain>/<resource>/` following the [file conventions](STYLE_GUIDE.md#resource-package-file-conventions).
4. Register the resource in `internal/provider/provider.go` (`Resources()`, `DataSources()`, `ListResources()`, or `Actions()` as applicable).
5. Add unit tests in the same package: `schema_test.go`, `input_builders_test.go`, `state_builders_test.go`, plus helpers/upgrader tests where relevant.
6. Add `resource_acceptance_test.go` (or `datasource_acceptance_test.go` for data-source-only packages) with `//go:build acceptance` on line 1. Use factories from `internal/testhelpers` (`AccPreCheck`, `AccTestProtoV6ProviderFactories`, `NewAcceptanceClient`, `RequireSmartGroupFixture`).
7. Add example `.tf` files under the matching `examples/` subdirectory (`examples/resources/<name>/`, `examples/data-sources/<name>/`, `examples/list-resources/<name>/`, `examples/actions/<name>/`).
8. Run `make generate` to regenerate copyright headers, format examples, and rebuild `docs/`.

See [TESTING.md](TESTING.md) for full testing guidance.

## Adding a Jamf Pro Resource

Jamf Pro resources (sourced from the `pro/` or `proclassic/` packages of `jamfplatform-go-sdk`) follow the same file conventions as Platform Services resources **plus** a planning gate. Use this workflow for every Jamf Pro construct.

1. **Pick the API namespace** from `JAMF_PRO_INVENTORY.md` (gitignored, at repo root). Update its status to `in-design`.
2. **Audit `pro/` vs `proclassic/`** for the namespace. Default to `pro/`. Switch to `proclassic/` only when `pro/` is missing or materially less feature-complete.
3. **Produce a one-page comparison** in the PR description: SDK package, function set proposed (CRUD + helpers), Terraform construct name (default: derived from SDK filename per [STYLE_GUIDE.md §Jamf Pro Resource Naming](STYLE_GUIDE.md#jamf-pro-resource-naming); override only if needed), **endpoint shape classification** (resource / data source / singleton / action — see [STYLE_GUIDE.md §Endpoint shape classification](STYLE_GUIDE.md#endpoint-shape-classification)), schema sketch, examples of similar shipped resources.
4. **Maintainer approval** locks in: SDK function set, Terraform construct name (and any override), target domain folder under `internal/resources/pro/<domain>/`, and the **minimum Jamf Pro version** (`minJamfProVersion` const — see [STYLE_GUIDE.md §Minimum Jamf Pro version check](STYLE_GUIDE.md#minimum-jamf-pro-version-check)). Source the version from Jamf release notes, the SDK function's `// Available since` comment, or hand-research; record it in `JAMF_PRO_INVENTORY.md`. Mark the row `in-progress`.
5. **Build the package** at `internal/resources/pro/<domain>/<resource>/` per the [file conventions](STYLE_GUIDE.md#resource-package-file-conventions). Mirror `internal/resources/pro/inventory/category/` for the Pro construct template (or `internal/resources/blueprints/blueprint/` for complex Platform Services shapes). Resource / data source / list resource / action Configure **must** funnel through `providerdata.ConfigurePro(ctx, req.ProviderData, minJamfProVersion, "jamfplatform_pro_<name>")` — do not hand-roll the type-assertion / version-fetch / version-gate / floor-warning boilerplate (see [STYLE_GUIDE.md §Pro Configure](STYLE_GUIDE.md#pro-configure-use-the-providerdataconfigurepro-helper)). Credentials are the same `JAMFPLATFORM_*` set used by Platform Services resources — there is no separate Pro credentials gate. **`crud.go` must carry the SDK-endpoints annotation block** at the top of the file with `Status: current. Last reviewed YYYY-MM-DD.` See [STYLE_GUIDE.md §Endpoint adoption & migration policy](STYLE_GUIDE.md#endpoint-adoption--migration-policy).
6. **Reuse shared schemas only when they exist**. Do **not** extract shared schemas (scope, site, category, criteria) preemptively — the trigger is 3 verified-identical SDK shapes across shipped resources. See [STYLE_GUIDE.md §Shared schemas (deferred abstraction)](STYLE_GUIDE.md#shared-schemas-deferred-abstraction). When the trigger fires, extract into `internal/common/schemas/` in a dedicated refactor PR.
7. **Tests + examples + docs**: same as a Platform Services resource — schema/input/state/upgrader unit tests, `resource_acceptance_test.go` with `//go:build acceptance` and `internal/testhelpers` factories, examples under `examples/resources/<name>/` (or the matching subdirectory), then `make generate`.
8. **Update inventory** to `shipped` after merge.

## Adding a New Data Source

Follow the same pattern as resources, but implement `datasource.DataSource` instead of `resource.Resource`. Data source packages that are standalone (not part of a CRUD resource) use `model_types.go` for their model structs.

## Project Structure

See `CLAUDE.md` for the full project structure and conventions. Key directories:

| Directory | Purpose |
|-----------|---------|
| `internal/provider/` | Provider configuration and resource registration |
| `internal/resources/` | Resource, data source, and list resource implementations |
| `internal/common/` | Shared helpers and RSQL filter utilities |
| `internal/actions/` | Fire-and-forget device management commands |
| `internal/testhelpers/` | Acceptance test utilities (provider factories, mock server, fixtures) |
| `examples/` | Example `.tf` configurations (resources, data-sources, list-resources, actions, provider) |
| `docs/` | Auto-generated provider documentation |
| `tools/` | `go:generate` entrypoint for `copywrite`, `terraform fmt`, and `tfplugindocs` |

The Jamf Platform API client lives in the external SDK `github.com/Jamf-Concepts/jamfplatform-go-sdk` — not in this repo.

## Dependencies

See [STYLE_GUIDE.md §Dependencies](STYLE_GUIDE.md#dependencies) for the allowed dependency set. In short: Go stdlib, `golang.org/x`, the HashiCorp Terraform Plugin family, and `jamfplatform-go-sdk`. Do not introduce other third-party dependencies without prior discussion.

## Release Versioning

The provider does **not** follow strict semver. Until further notice:

- **Patch** (`X.Y.Z+1`): bug fixes only, no schema or behavior changes.
- **Minor** (`X.Y+1.0`): everything else — new resources, new attributes, deprecations, **and breaking changes** (renames, removals, attribute-type changes, behavior changes).
- **Major** (`X+1.0.0`): reserved. Bumped only as a deliberate coordinated cleanup release (e.g., wide-scale removals or restructuring). Not required for individual breaking changes.

Document any breaking change explicitly in the release notes (generated via goreleaser — see [TESTING.md](TESTING.md) and the release workflow). When the policy changes (e.g., on adopting strict semver post-Pro-rollout), update this section.

## Commit Messages

Use [conventional commit](https://www.conventionalcommits.org/) style messages:

- `feat: add device_group import support`
- `fix: handle nil ODV in benchmark rules`
- `test: add schema validation for blueprint components`
- `refactor: extract common polling logic to helpers`
- `chore: update CI workflow action versions`
- `docs: add TESTING.md`

## Pull Requests

- Keep PRs focused — one feature or fix per PR.
- Include unit tests for new code.
- Include acceptance tests for new resources and data sources.
- Update `examples/` for new Terraform constructs.
- Run `make generate` if schema descriptions changed (to update docs and copyright headers).
- CI must pass before merge.

## Reporting Issues

Open an issue on GitHub with:

- Provider version and Terraform version.
- Relevant Terraform configuration (redact credentials).
- Expected vs actual behaviour.
- Any error messages or logs.
