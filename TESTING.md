# Testing

This document covers the testing strategy and instructions for the Terraform Provider for Jamf Platform.

## Test Categories

| Category   | Build Tag    | Requires API | Command                                      |
|------------|--------------|--------------|----------------------------------------------|
| Unit       | (none)       | No           | `make test`                                  |
| Acceptance | `acceptance` | Yes          | `make testacc`                               |

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

Test files live alongside the code they test, following Go convention. The Jamf Platform API client lives in the external SDK `jamfplatform-go-sdk` and its tests live in that repo — they are not part of this provider's test suite.

```
internal/
├── common/
│   ├── helpers/
│   │   ├── helpers_test.go
│   │   └── dynamic_json_test.go
│   └── filters/filters_test.go
├── testhelpers/                          # Acceptance fixtures (mock server, real-client builder, smart group fixture)
├── actions/device/schema_test.go
├── resources/
│   ├── blueprints/blueprint/
│   │   ├── schema_test.go                # Unit: schema validation
│   │   ├── helpers_test.go               # Unit: helpers
│   │   ├── input_builders_test.go        # Unit: API input builders
│   │   ├── state_builders_test.go        # Unit: API → state mapping
│   │   ├── state_upgrader_test.go        # Unit: schema migrations
│   │   └── resource_acceptance_test.go   # Acceptance: full CRUD
│   ├── cbengine/benchmark/
│   │   ├── schema_test.go
│   │   ├── input_builders_test.go
│   │   ├── state_builders_test.go
│   │   └── resource_acceptance_test.go
│   ├── device_group/
│   │   ├── schema_test.go
│   │   ├── helpers_test.go
│   │   ├── input_builders_test.go
│   │   ├── state_builders_test.go
│   │   ├── state_upgrader_test.go
│   │   └── resource_acceptance_test.go
│   └── devices/
│       ├── schema_test.go
│       └── datasource_acceptance_test.go # Acceptance: data-source-only package
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

### Jamf Pro fixtures (forthcoming)

When the first Jamf Pro resource lands (`internal/resources/pro/...`), `internal/testhelpers` will gain Pro-specific fixtures. The acceptance client itself is unchanged — Pro resources use the same `*jamfplatform.Client` built from the same `JAMFPLATFORM_*` credentials as Platform Services resources.

**Tenant data isolation conventions for Pro acceptance tests:**

- Use the existing `tf-acc-` prefix on every resource created (matches STYLE_GUIDE convention).
- Pro resources have richer dependency graphs (a policy needs a category, smart group, script, package). Use `t.Cleanup` for dependency-ordered teardown — register cleanups in reverse-creation order so dependencies are torn down before their dependents.
- Shared fixtures that span multiple test files (e.g., a long-lived test category, a known smart group) live in `internal/testhelpers` and are cleaned up via `TestMain` — same pattern as `RequireSmartGroupFixture`. Add new `Require*Fixture` helpers as needed.
- Tests must be independent — never assume another test has run first.

**CI scaling plan** (deferred until Pro suite exists):

The current acceptance job in `.github/workflows/integration-tests.yml` has a 30-minute timeout and runs Platform Services tests serially (`-p=1`). As the Pro suite grows, plan is to **split acceptance into a dedicated workflow** (`acceptance-tests.yml`) — a single serial job (no matrix; parallel acceptance against the same tenant causes naming/ID collisions). The split also lets us tune the Pro job's timeout independently and run it on a different cadence (e.g., manual-only or post-merge) if needed.

### Benchmark-specific considerations

CBEngine benchmarks deploy asynchronously. The benchmark must reach `SYNCED` state before it can be safely deleted — deleting while in `PENDING` state causes the benchmark to get stuck in `DELETING`. Use `testhelpers.EnsureBenchmarkDeletedByID` or `testhelpers.EnsureBenchmarkDeleted` for cleanup, which handle sync-waiting automatically.

## CI/CD

All CI lives in **`.github/workflows/integration-tests.yml`**, triggered on PRs to `main` and via `workflow_dispatch`.

| Job          | What it does                                            | Gating                                  | Timeout |
|--------------|---------------------------------------------------------|-----------------------------------------|---------|
| `build`      | `go build` + `golangci-lint run`                        | —                                       | 5 min   |
| `generate`   | Runs `make generate`, fails if `git diff` is non-empty  | —                                       | default |
| `unit`       | `go test -v -cover -count=1 ./...`                      | Needs `build`                           | 10 min  |
| `acceptance` | `go test -v -cover -count=1 -tags=acceptance -p=1 ./...` | Needs `unit`; gated by `acceptance` env | 30 min  |

The `acceptance` job requires manual approval through the GitHub `acceptance` environment before it runs. It executes against Terraform `1.14.*`.

### Required GitHub Secrets

Bound to the `acceptance` environment:

| Secret                       | Description                                                   |
|------------------------------|---------------------------------------------------------------|
| `JAMFPLATFORM_BASE_URL`      | Jamf Platform tenant URL                                      |
| `JAMFPLATFORM_CLIENT_ID`     | OAuth client ID                                               |
| `JAMFPLATFORM_CLIENT_SECRET` | OAuth client secret                                           |
| `JAMFPLATFORM_TENANT_ID`     | Tenant ID (consumed by acceptance fixtures where applicable)  |

## Terraform-Native Tests (Legacy / Supplementary)

The `testing/` directory contains Terraform-native integration tests (`.tftest.hcl`, `.tfquery.hcl`) that use `terraform test` and `terraform query`. These have been superseded by the Go acceptance suite (now the gating CI suite) and are kept as a supplementary check only. They are **not** part of the CI pipeline. See [testing/README.md](testing/README.md) for usage.
