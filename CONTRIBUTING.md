# Contributing

Thank you for your interest in contributing to the Terraform Provider for Jamf Platform.

## Prerequisites

- **Go** >= 1.25 (see `go.mod` for the exact version)
- **Terraform** >= 1.0
- **golangci-lint** for linting
- A Jamf Platform tenant with API credentials (for acceptance tests only)

## Getting Started

```bash
# Clone the repository
git clone https://github.com/Jamf-Concepts/terraform-provider-jamfplatform.git
cd terraform-provider-jamfplatform

# Install dependencies
go mod download

# Build
go build -v ./...

# Run unit tests
go test -v -cover -count=1 ./...
```

## Development Workflow

1. Create a feature branch from `main`.
2. Make your changes following the [style guide](STYLE_GUIDE.md).
3. Run formatting, linting, and tests before committing:

   ```bash
   go fmt ./...
   golangci-lint run
   go test -v -cover -count=1 ./...
   ```

4. Open a pull request against `main`. CI will run build, lint, and unit tests automatically.
5. If your changes affect provider behaviour, request an acceptance test run (manual, approval-gated).

## Adding a New Resource

1. Obtain the OpenAPI specification or request/response examples for the Jamf Platform endpoint.
2. Add client methods in `internal/client/` with versioned function names (e.g. `CreateMyResourceV1`).
3. Create the resource package under `internal/resources/<domain>/<resource>/` following the [file conventions](STYLE_GUIDE.md#resource-package-file-conventions).
4. Register the resource in `internal/provider/provider.go` in the `Resources()` method.
5. Add unit tests:
   - `schema_test.go` for schema validation.
   - Client mock tests in `internal/client/`.
6. Add acceptance tests in `resource_acceptance_test.go` with the `//go:build acceptance` tag.
7. Add example configurations in `examples/`.
8. Run `go generate ./...` from the `tools/` directory to regenerate documentation.

See [TESTING.md](TESTING.md) for full testing guidance.

## Adding a New Data Source

Follow the same pattern as resources, but implement `datasource.DataSource` instead of `resource.Resource`. Data source packages that are standalone (not part of a CRUD resource) use `model_types.go` for their model structs.

## Project Structure

See `CLAUDE.md` for the full project structure and conventions. Key directories:

| Directory | Purpose |
|-----------|---------|
| `internal/provider/` | Provider configuration and resource registration |
| `internal/resources/` | Resource, data source, and list resource implementations |
| `internal/client/` | Jamf Platform API client |
| `internal/common/` | Shared helpers and RSQL filter utilities |
| `internal/actions/` | Fire-and-forget device management commands |
| `internal/testhelpers/` | Acceptance test utilities |
| `examples/` | Example `.tf` configurations |
| `docs/` | Auto-generated provider documentation |

## Dependencies

This project only uses native Go, `golang.org/x` packages, and Terraform Plugin Framework packages. Do not introduce third-party dependencies without discussion.

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
- Run `go generate` if schema descriptions changed (to update docs).
- CI must pass before merge.

## Reporting Issues

Open an issue on GitHub with:

- Provider version and Terraform version.
- Relevant Terraform configuration (redact credentials).
- Expected vs actual behaviour.
- Any error messages or logs.
