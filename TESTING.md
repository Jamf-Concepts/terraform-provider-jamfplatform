# Testing

This document covers the testing strategy and instructions for the Terraform Provider for Jamf Platform.

## Test Categories

| Category | Count | Build Tag | Requires API | Command |
|----------|-------|-----------|--------------|---------|
| Unit | 369 | (none) | No | `go test ./...` |
| Acceptance | 66 | `acceptance` | Yes | `go test -tags=acceptance -p=1 ./...` |

### Unit Tests

Unit tests validate schema definitions, metadata, plan modifiers, state migration, type flatteners/expanders, helper functions, and client HTTP behaviour using a mock server. They do not require network access or API credentials.

```bash
go test -v -cover -count=1 ./...
```

### Acceptance Tests

Acceptance tests create, read, update, and delete real resources against a live Jamf Platform tenant. They are gated behind the `//go:build acceptance` build tag so they never run by default.

```bash
export JAMFPLATFORM_BASE_URL="https://us.apigw.jamf.com"
export JAMFPLATFORM_CLIENT_ID="your-client-id"
export JAMFPLATFORM_CLIENT_SECRET="your-client-secret"

go test -v -cover -count=1 -tags=acceptance -p=1 ./...
```

**Flags explained:**

- `-tags=acceptance` includes files with the `//go:build acceptance` tag.
- `-p=1` runs packages sequentially to avoid resource naming conflicts.
- `-count=1` bypasses the Go test cache (useful for re-runs).

## Test File Layout

Test files live alongside the code they test, following Go convention:

```
internal/
├── client/
│   ├── client_test.go                    # Unit: mock HTTP tests
│   ├── blueprints_acceptance_test.go     # Acceptance: real API
│   ├── cbengine_acceptance_test.go
│   ├── device_groups_acceptance_test.go
│   ├── datasources_acceptance_test.go
│   └── main_acceptance_test.go           # TestMain for fixture cleanup
├── common/
│   ├── helpers/helpers_test.go
│   └── filters/filters_test.go
├── resources/
│   ├── blueprints/blueprint/
│   │   ├── schema_test.go                # Unit: schema validation
│   │   ├── helpers_test.go               # Unit: component helpers
│   │   └── resource_acceptance_test.go   # Acceptance: full CRUD
│   ├── cbengine/benchmark/
│   │   ├── schema_test.go
│   │   └── resource_acceptance_test.go
│   └── device_group/
│       ├── schema_test.go
│       ├── helpers_test.go
│       └── resource_acceptance_test.go
└── ...
```

### Naming conventions

- `*_test.go` — unit tests (no build tag).
- `*_acceptance_test.go` — acceptance tests (with `//go:build acceptance`).
- `schema_test.go` — schema validation tests present in every resource/data source package.

## Writing Unit Tests

Unit tests use Go's standard `testing` package. Client tests use `httptest.NewServer` to mock the Jamf Platform API.

```go
func TestMyFunction(t *testing.T) {
    // arrange, act, assert
}
```

Schema validation tests use the `terraform-plugin-testing` framework:

```go
func TestSchemaValidity_Resource(t *testing.T) {
    ctx := context.Background()
    schemaResp := resource.SchemaResponse{}
    NewResource().Schema(ctx, resource.SchemaRequest{}, &schemaResp)
    assert.False(t, schemaResp.Diagnostics.HasError())
}
```

## Writing Acceptance Tests

Every acceptance test file must include the build tag:

```go
//go:build acceptance

package mypackage_test
```

### Resource acceptance tests

Resource acceptance tests use the `terraform-plugin-testing` framework with `resource.TestCase`:

```go
func TestAccResource_MyResource(t *testing.T) {
    testhelpers.AccPreCheck(t)

    resource.Test(t, resource.TestCase{
        ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
        CheckDestroy:             testAccCheckMyResourceDestroy(t),
        Steps: []resource.TestStep{
            {
                Config: `resource "jamfplatform_my_resource" "test" { ... }`,
                Check: resource.ComposeAggregateTestCheckFunc(
                    resource.TestCheckResourceAttrSet("jamfplatform_my_resource.test", "id"),
                ),
            },
        },
    })
}
```

**Requirements:**

- Call `testhelpers.AccPreCheck(t)` at the start of every test.
- Always provide a `CheckDestroy` function that verifies resources are removed after the test.
- Use unique resource names per test to avoid conflicts (e.g. include the test name or a suffix).

### Client acceptance tests

Client-level acceptance tests call API methods directly to validate the client layer:

```go
func TestAcceptance_MyResource_Create(t *testing.T) {
    c := testhelpers.NewAcceptanceClient(t)
    ctx := context.Background()
    // create, verify, cleanup via t.Cleanup
}
```

### Shared fixtures

The `testhelpers.RequireSmartGroupFixture(t)` function provides a shared smart group for tests that need a device group target. The fixture is created once per test binary and cleaned up via `TestMain`.

### Benchmark-specific considerations

CBEngine benchmarks deploy asynchronously. The benchmark must reach `SYNCED` state before it can be safely deleted — deleting while in `PENDING` state causes the benchmark to get stuck in `DELETING`. Use `testhelpers.EnsureBenchmarkDeletedByID` or `testhelpers.EnsureBenchmarkDeleted` for cleanup, which handle sync-waiting automatically.

## CI/CD

### Integration Tests (automated)

Runs on every PR to `main` and every push to `main`. Covers build, lint, and unit tests.

Workflow: `.github/workflows/integration-test.yml`

| Job | What it does | Timeout |
|-----|--------------|---------|
| `build` | `go build` + `golangci-lint run` | 5 min |
| `generate` | Validates generated docs are up to date | 5 min |
| `unit` | `go test -v -cover -count=1 ./...` | 10 min |

### Acceptance Tests (manual, approval-gated)

Triggered manually via `workflow_dispatch`. Requires approval through the GitHub `acceptance` environment.

Workflow: `.github/workflows/acceptance-tests.yml`

Runs against a matrix of Terraform versions (1.13.x, 1.14.x) with credentials from repository secrets.

### Required GitHub Secrets

| Secret | Description |
|--------|-------------|
| `JAMFPLATFORM_BASE_URL` | Jamf Platform tenant URL |
| `JAMFPLATFORM_CLIENT_ID` | OAuth client ID |
| `JAMFPLATFORM_CLIENT_SECRET` | OAuth client secret |

## Terraform-Native Tests (Legacy)

The `testing/` directory contains Terraform-native integration tests (`.tftest.hcl`, `.tfquery.hcl`) that use `terraform test` and `terraform query`. See [testing/README.md](testing/README.md) for details. These are supplementary to the Go acceptance tests and are not part of the CI pipeline.
