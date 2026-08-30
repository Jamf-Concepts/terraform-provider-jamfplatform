# Testing

This document covers the testing strategy and instructions for the Terraform Provider for Jamf Platform.

## Test Categories

| Category               | Build Tag    | Requires API | Command                                                                                       |
|------------------------|--------------|--------------|-----------------------------------------------------------------------------------------------|
| Unit                   | (none)       | No           | `make test`                                                                                   |
| Unit (tooling)         | `acctargets` | No           | `make test-scripts` — `scripts/acctargets`, which `go test ./...` cannot see behind its build tag |
| Acceptance (all)       | `acceptance` | Yes          | `make testacc`                                                                                |
| Acceptance (changed)   | `acceptance` | Yes          | `make testacc-changed` (changed packages + their transitive dependents)                       |
| Acceptance (targeted)  | `acceptance` | Yes          | `make testacc-run RUN=<regex> PKG=<package>` (e.g. `RUN=TestAccResource_ProSite_Basic PKG=./internal/resources/pro/site/...`) |
| Acceptance (functions) | `acceptance` | No           | `make testacc-run RUN=TestAccFunction PKG=./internal/functions/...` — provider functions are offline; their tests need a Terraform binary but no tenant (they call `testhelpers.AccPreCheckOffline`, which sets `TF_ACC` without the credential gate) |

### Unit Tests

Unit tests validate schema definitions, metadata, plan modifiers, state migration, type flatteners/expanders, helper functions, and client HTTP behaviour using a mock server. They do not require network access or API credentials.

```bash
go test -v -cover -count=1 ./...
```

### Acceptance Tests

Acceptance tests create, read, update, and delete real resources against a live Jamf Platform tenant. They are gated behind the `//go:build acceptance` build tag so they never run by default.

```bash
export JAMFPLATFORM_BASE_URL="https://us.api.jamfcloud.com"
export JAMFPLATFORM_CLIENT_ID="your-client-id"
export JAMFPLATFORM_CLIENT_SECRET="your-client-secret"
export JAMFPLATFORM_ENVIRONMENT_ID="your-environment-id" # preferred; or JAMFPLATFORM_TENANT_ID (legacy), never both

# Jamf Security Cloud tests additionally require you to DECLARE that the scope above
# is a Security Cloud one. The value must match the scope actually configured; unset
# or mismatched and every Security Cloud test skips rather than failing.
export JAMFPLATFORM_SECURITY_CLOUD_ENVIRONMENT_ID="$JAMFPLATFORM_ENVIRONMENT_ID"

go test -v -cover -count=1 -tags=acceptance -p=1 ./...
```

**Flags explained:**

- `-tags=acceptance` includes files with the `//go:build acceptance` tag.
- `-p=1` runs packages sequentially to avoid resource naming conflicts.
- `-count=1` bypasses the Go test cache (useful for re-runs).

**Targeted re-runs:** `make testacc-run` accepts `RUN=<Go -run regex>` and `PKG=<package path>` overrides; extra flags can be appended via `TESTARGS`. Example:

```bash
make testacc-run \
  RUN=TestAccResource_ProNetworkSegment_Basic \
  PKG=./internal/resources/pro/network_segment/...
```

**Changed-scope runs:** `make testacc-changed` runs acceptance tests only for the
packages your branch touches plus everything that transitively depends on them.
It shells out to `scripts/acctargets`, which diffs against `origin/main` (override
with `BASE=<ref>`) and walks the package import graph:

- Change one resource (e.g. `internal/resources/pro/category/`) → only that
  package's acceptance tests run.
- Change a shared helper (e.g. `internal/common/scope/`) → that package **and**
  every consumer (`policy`, the apps, `user_group`, the VPP resources, the
  config profiles, the advanced searches, …) run.
- **Add** to a hub package — a new `providerdata` helper, a new `testhelpers`
  fixture, a new shared validator — → only that package. Changed Go files are
  compared declaration by declaration, and no unchanged package can reference a
  declaration that did not exist at the base ref. Any package that gained such a
  reference changed too, and selects itself.
- **Modify or remove** an existing declaration in one of those hubs → the full
  fan-out, because that genuinely can reach every consumer.
- Register or unregister a construct in `internal/provider/provider.go` → only
  the packages whose constructors moved. Any other edit to that file (a
  `Configure` change, a new provider attribute) falls back to the full fan-out.
  Reordering a list changes no constructor and so selects nothing.
- Change a global file — `go.mod`/`go.sum`, the `GNUmakefile`, anything under
  `.github/workflows/`, or `scripts/acctargets/` itself — resolves to the full
  suite (`./...`).

Every uncertainty resolves to the wider answer: an unparseable revision, a
declaration that moved, a registration list that is no longer a plain list. Over-
running costs time; under-running costs a regression. `make test-scripts` runs the
tool's own unit tests, which `go test ./...` cannot see (the tool is behind the
`acctargets` build tag so it stays out of the provider binary).

The graph deliberately **cuts `internal/provider`'s out-edges**. The provider
imports every resource package purely to register it, and `testhelpers` imports
the provider, so without the cut every change would look like it touches the
whole suite. The cut keeps real edges (a package's own deps, shared helpers,
cross-package test fixtures) and drops only the registration fan-out.

## Test File Layout

Test files live alongside the code they test, following Go convention. The Jamf Platform API client lives in the external SDK `jamfplatform-go-sdk` and its tests live in that repo — they are not part of this provider's test suite.

```
internal/
├── common/
│   ├── helpers/                          # helpers_test.go, dynamic_json_test.go, ids_test.go, pro_version_test.go
│   ├── filters/filters_test.go
│   └── scope/                            # builders_test.go, schema_test.go, validators_test.go
├── testhelpers/                          # Acceptance fixtures (mock server, real-client builder, shared fixtures)
├── actions/device/schema_test.go
├── functions/                            # Provider-defined functions (offline — no tenant needed)
│   ├── mobileconfig/                     # assemble_test, function_test (Run seam via real types.Dynamic), function_acceptance_test
│   └── mcx_forced_payload/               # render_test, function_test, function_acceptance_test
├── resources/
│   ├── blueprints/blueprint/             # schema_test, helpers_test, input_builders_test, state_builders_test, state_upgrader_test, resource_acceptance_test
│   ├── cbengine/benchmark/               # schema_test, input_builders_test, state_builders_test, resource_acceptance_test
│   ├── device_group/                     # schema_test, helpers_test, input_builders_test, state_builders_test, state_upgrader_test, resource_acceptance_test
│   ├── devices/                          # schema_test, datasource_acceptance_test (data-source-only)
│   └── pro/                              # Same file shapes per resource, plus marshal_smoke_test for ProClassic XML resources
│       ├── inventory/<resource>/         # category, site, building, department, network_segment, ibeacon, dock_item, directory_binding, disk_encryption_configuration, package, icon, printer
│       ├── policies/{policy,script}/
│       ├── configuration_profiles/macos_configuration_profile/  # + helpers_corpus_test (build tag profile_corpus)
│       ├── settings/self_service_plus_settings/
│       └── users/user_group/
```

Each leaf folder follows the `*_test.go` (unit) + `resource_acceptance_test.go` / `datasource_acceptance_test.go` (acceptance, `//go:build acceptance`) split. Acceptance file names are flat — no per-scenario splits.

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

**Coverage requirements (every new resource):**

A single Create-only test is **not** sufficient. The acceptance suite for a resource must collectively exercise:

1. **An Update round-trip — required.** At least one test must drive a multi-step `resource.TestCase` that mutates the resource in place. Every Optional/Required attribute that is *not* `RequiresReplace` must be exercised across steps, including:
   - At least one nested-block / list-attribute element add **and** remove (proves list reconciliation, not just element-content mutation).
   - At least one server-derived computed sibling (hash, URL, epoch) re-resolves correctly after the change.
   - This is the highest-value test in the file — it catches `mergePlanIntoServerState`, `UseStateForUnknown`, and lossy-PUT regressions that single-step Create tests cannot.
2. **A negative case per declared cross-field validator.** Every `ConflictsWith`, `AlsoRequires`, `OneOf`, and custom plan-time validator on the resource needs an `ExpectError` test. The positive case alone is not enough — silent validator removal is the easiest regression to ship.
3. **An import round-trip** via `ImportStateVerify: true`. List `ImportStateVerifyIgnore` for any attribute the importer legitimately cannot recover (e.g. write-only inputs, local-path fields) and justify each entry in a code comment.
4. **All "shape" branches** — for any resource with mutually exclusive sub-blocks (auth modes, scope variants, payload kinds), include at least one happy-path test per shape so each input-builder branch is wire-validated.
5. **Server-side drift recovery** where applicable. For self-healing resources (image uploads, hash convergence, lossy GETs), include a multi-step test that mutates a user input and asserts the Computed echo attributes update — not just that they remain `Set`.

The reference enumeration of these patterns lives in `internal/resources/pro/enrollment_customization/resource_acceptance_test.go`. Copy from there when scaffolding a new resource's acceptance file.

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

### Jamf Pro acceptance tests

Pro resources use the same `*jamfplatform.Client` and the same `JAMFPLATFORM_*` credentials as Platform Services resources. Tenant data isolation conventions:

- Use the `tf-acc-` prefix on every resource created.
- Pro resources have richer dependency graphs (a policy needs a category, smart group, script, package). Use `t.Cleanup` for dependency-ordered teardown — register cleanups in reverse-creation order.
- Shared fixtures that span multiple test files live in `internal/testhelpers` and are cleaned up via `TestMain` (precedent: `RequireSmartGroupFixture`). Add new `Require*Fixture` helpers as needed.
- Tests must be independent — never assume another test has run first.

**Acceptance test files MUST declare `//go:build acceptance` on line 1** or they leak into the unit run.

### Tenant-prerequisite fixtures (server-state gates)

Some Pro features can only be created when the tenant is in a prerequisite state — e.g. an enrollment-customization **SSO pane** requires SAML configured (`403 [INVALID_STATE] : SAML must be configured`), and a `DIRECTORY_SERVICE_ATTRIBUTE_MAPPING` extension attribute requires LDAP configured (`400 [INVALID_CONTENT] ... if LDAP is not configured`). Stand the prerequisite up as an in-config **fixture resource the subject `depends_on`**, so the apply orders the prerequisite first. Two flavours:

- **Dummy fixture, no gating** — when the prerequisite resource doesn't validate live connectivity, build a placeholder. `jamfplatform_pro_ldap_server` does not verify the LDAP connection, so a dummy (`Open Directory`, fake `ldap.acc-anon.example.com:389`, `authentication_type = none`) configures LDAP with no real server and **needs no env var**. Reference: `ceaDSAM`/`mdeaDSAM`.
- **Real-credential fixture, env-gated** — when the prerequisite needs real external infra (a SAML IdP metadata URL), gate the test on an env var and **skip** when unset (`t.Skipf`). Reference: the enrollment-customization SSO-pane tests gate on `JAMFPLATFORM_ACC_SSO_IDP_URL`; the `sso_settings` suite owns the same var.
- **Pre-existing tenant object, env-gated** — when the prerequisite is an object the provider *cannot* create, name it by env var and skip when unset. `jamfplatform_pro_pki_adcs` in `OUTBOUND` mode needs a Jamf Pro API client, and API clients and roles were withdrawn from the Platform API at GA (they are created in Jamf Account), so the OUTBOUND tests take the client's UUID from `JAMFPLATFORM_ACC_ADCS_API_CLIENT_ID` and skip without it.

Two cautions: (1) prefer a fixture that **mutates the tenant minimally and reversibly** — e.g. SSO uses `OIDC_WITH_SAML`, not pure SAML, so Jamf ID admin login keeps working. (2) Many prerequisite singletons (`sso_settings`, `computer_inventory_collection_settings`) have **state-only deletes**, so teardown leaves the tenant in the fixture's state — idempotent on re-run, but document it and don't let other suites assume a clean SSO/inventory baseline.

### Do not `ImportStateVerify` async server-computed values

A `Computed` attribute that the server populates **asynchronously** (e.g. `computer_prestage_enrollment.profile_uuid` — the "information out of date" window) can read empty on a plain `GET` even on a settled object, so it does not reliably round-trip on import. Add it to `ImportStateVerifyIgnore` (alongside `timeouts` and WriteOnly secrets). It is server-generated, not user-settable, so verifying it on import adds no coverage. **Do not** try to force it to populate inside `Read` (e.g. a readiness poll) — for a GET-sensitive value that can make refresh worse (a transient empty becomes a hard timeout error).

### Profile-corpus regression test (opt-in build tag)

`internal/resources/pro/macos_configuration_profile/helpers_corpus_test.go` is gated by `//go:build profile_corpus`. It iterates a 200-profile mobileconfig corpus under `testing/profile_roundtrip/` (gitignored, developer-machine-only) and asserts the mask-and-compare diff suppression is stable. Run with:

```bash
go test -tags profile_corpus ./internal/resources/pro/macos_configuration_profile/...
```

Not part of CI. Regenerate the corpus before running by replaying `/tmp/sample_titles.py` + `/tmp/roundtrip.py` against a tenant. Background on the diff classes the mask covers: [STYLE_GUIDE.md §Configuration profile payload diff suppression](STYLE_GUIDE.md#configuration-profile-payload-diff-suppression-mask-and-compare).

### CI scaling

The full suite runs serially (`-p=1`) against a single shared tenant, which can
exceed two hours end to end. To keep PRs fast, CI scopes acceptance to the
change set:

- **Per PR** (`integration-tests.yml`): the `plan-acc` job runs
  `scripts/acctargets` to compute the affected packages, then calls the reusable
  `acceptance.yml` workflow with just that subset. A PR touching no acceptance
  package skips the acceptance job entirely.
- **Nightly + on demand** (`acceptance-full.yml`): the complete suite (`./...`)
  as a regression safety net, on a `0 2 * * *` cron and `workflow_dispatch`.

Both paths run through the reusable `acceptance.yml` (one copy of the tenant
secret wiring) and share the `acceptance-tenant` concurrency group, so a
scheduled full run queues behind any in-progress PR run rather than colliding on
the tenant. Acceptance against one tenant must stay serial — parallel runs cause
naming/ID collisions — so scaling is "test fewer packages," not "test in
parallel."

### Benchmark-specific considerations

CBEngine benchmarks deploy asynchronously. The benchmark must reach `SYNCED` state before it can be safely deleted — deleting while in `PENDING` state causes the benchmark to get stuck in `DELETING`. Use `testhelpers.EnsureBenchmarkDeletedByID` or `testhelpers.EnsureBenchmarkDeleted` for cleanup, which handle sync-waiting automatically.

## CI/CD

PR CI lives in **`.github/workflows/integration-tests.yml`**, triggered on PRs to `main` and via `workflow_dispatch`. The acceptance run itself is factored into the reusable **`.github/workflows/acceptance.yml`**, and the full-suite schedule lives in **`.github/workflows/acceptance-full.yml`**.

| Job          | What it does                                                              | Gating                                  | Timeout |
|--------------|---------------------------------------------------------------------------|-----------------------------------------|---------|
| `build`      | `go build` + `golangci-lint run`                                          | —                                       | 5 min   |
| `generate`   | Runs `make generate`, fails if `git diff` is non-empty                    | —                                       | default |
| `unit`       | `go test -v -cover -count=1 ./...`                                        | Needs `build`                           | 10 min  |
| `plan-acc`   | `scripts/acctargets` → the affected package set for this PR               | Needs `unit`                            | 5 min   |
| `acceptance` | Calls `acceptance.yml` with the scoped package set (`-p=1`)               | Needs `plan-acc`; skipped if empty; gated by `acceptance` env | 6 h (default) |

The reusable `acceptance.yml` also backs the scheduled full run (`acceptance-full.yml`, `packages: ./...`). Both run in the GitHub `acceptance` environment (which holds the tenant secrets) and execute against Terraform `1.15.*`.

### Required GitHub Secrets

Bound to the `acceptance` environment:

| Secret                       | Description                                                   |
|------------------------------|---------------------------------------------------------------|
| `JAMFPLATFORM_BASE_URL`      | Jamf Platform tenant URL                                      |
| `JAMFPLATFORM_CLIENT_ID`     | OAuth client ID                                               |
| `JAMFPLATFORM_CLIENT_SECRET` | OAuth client secret                                           |
| `JAMFPLATFORM_ENVIRONMENT_ID`| Platform environment ID — preferred scope; mutually exclusive with the next row |
| `JAMFPLATFORM_TENANT_ID`     | Tenant UUID — legacy scope; set exactly one of these two                        |
| `JAMFPLATFORM_SECURITY_CLOUD_ENVIRONMENT_ID` | Declares that the configured `JAMFPLATFORM_ENVIRONMENT_ID` belongs to a Jamf Security Cloud tenant. Must equal it. Unset or mismatched → every Security Cloud test skips |
| `JAMFPLATFORM_SECURITY_CLOUD_TENANT_ID` | Same, for `JAMFPLATFORM_TENANT_ID`. Set at most one of these two. Also names the tenant a ZTNA gateway grants access to — the gateway tests skip without it, because `tenantIds` is required on every gateway and is validated against the caller's organization |

### Jamf Security Cloud coverage is opt-in, and partly tenant-scope-only

Neither declaration variable is set in CI, so **every Jamf Security Cloud acceptance
test skips there** — 48 of them, each printing the reason. The suite is green and the
coverage is zero; do not read one as the other. They are exercised by running locally
against an entitled tenant.

Turning them on is just setting `JAMFPLATFORM_SECURITY_CLOUD_TENANT_ID` to the same
value as `JAMFPLATFORM_TENANT_ID`. The gate requires the two to match, so a value
left over from a different tenant skips rather than running against the wrong estate.

**Under an environment-scoped integration the ZTNA gateway tests cannot run at all.**
`tenantIds` is required on every gateway and grouped gateway and is validated against
the caller's organization, and nothing in the API exposes the tenants belonging to an
environment — so there is no id for a test to name. `RequireSecurityCloudTenantID`
skips rather than guessing. Custom DNS zones and the shared-gateway catalogue have no
such field and are unaffected.

Also untested: the Security Cloud surface under `X-Environment-Id`. Every wire probe
behind these resources used a tenant-scoped integration.
`providerdata.ConfigureSecurityCloud` admits both scopes on the strength of the spec,
not a probe. If `/securitycloud` turns out not to answer under an environment
header, the fix is to drop `ScopeEnvironment` from that call — the same one-token
narrowing the scope gate was built for.
