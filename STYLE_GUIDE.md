# Style Guide

Code style conventions for the Terraform Provider for Jamf Platform.

## Go Conventions

- Follow standard Go conventions and idiomatic patterns.
- Run `go fmt ./...` and `golangci-lint run` before committing.
- Use clear, descriptive names for variables, functions, and types.
- Every exported constant, function, variable set, and type must have a short comment describing its purpose.
- Do not add comments inside type definitions or function bodies.

## Dependencies

Allowed:

- Go standard library.
- `golang.org/x` packages.
- HashiCorp Terraform Plugin packages: `terraform-plugin-framework`, `terraform-plugin-framework-timeouts`, `terraform-plugin-framework-validators`, `terraform-plugin-go`, `terraform-plugin-log`, `terraform-plugin-testing`.
- `github.com/Jamf-Concepts/jamfplatform-go-sdk` — required for all Jamf Platform API access (auth, HTTP transport, request/response types).

Do not introduce other third-party dependencies without prior discussion.

## Resource Package File Conventions

Resource packages live under `internal/resources/<domain>/<resource>/` and use resource-agnostic filenames:

| File | Purpose |
|------|---------|
| `resource.go` | Schema definition and boilerplate |
| `crud.go` | Create, Read, Update, Delete, and ImportState |
| `model_types.go` | Terraform model structs |
| `schema_types.go` | Attribute type maps for `ObjectValue`/`ListValue` state |
| `mappings.go` | Lookup tables and name mappings |
| `input_builders.go` | Build API request payloads from Terraform model data |
| `state_builders.go` | Map API responses to Terraform state |
| `helpers.go` | Resource-specific helper functions |
| `state_upgrader.go` | Schema version upgraders (when bumping schema version) |
| `plan_modifiers.go` | Schema plan modifiers (if needed) |
| `validators.go` | Schema validators (if needed) |
| `list_resource.go` | List resource implementation |
| `data_source.go` | Data source implementation |

### Optional split-outs for complex resources

- `endpoints_builders.go` / `endpoints_state.go` — when endpoint logic dominates.
- `nested_builders.go` / `nested_state.go` — for large nested payloads.

### Data-source-only packages

Packages that only contain a data source use `model_types.go` for their model structs and `data_source.go` for the implementation.

## Test File Conventions

| File | Purpose |
|------|---------|
| `schema_test.go` | Schema validation (every package) |
| `helpers_test.go` | Helper function tests |
| `input_builders_test.go` | Input builder tests |
| `state_builders_test.go` | State builder tests |
| `state_upgrader_test.go` | State upgrader tests (where present) |
| `resource_acceptance_test.go` | Resource acceptance tests (`//go:build acceptance`) |
| `datasource_acceptance_test.go` | Data-source-only package acceptance tests (`//go:build acceptance`) |

## Client Conventions

The Jamf Platform API client lives in the external SDK `github.com/Jamf-Concepts/jamfplatform-go-sdk` (package `jamfplatform`). This repository imports it; it is not vendored. To consume new endpoints, bump the SDK dep in `go.mod` (or coordinate changes upstream first).

The SDK follows the conventions below — match them when contributing upstream.

### Versioned naming

Client types and functions use explicit version suffixes to support multiple API versions:

```go
// V1 functions
func (c *Client) CreateDeviceGroupV1(ctx context.Context, req *DeviceGroupCreateRepresentationV1) (*DeviceGroupCreateResponseV1, error)

// V2 functions added alongside V1
func (c *Client) CreateCBEngineBenchmarkV2(ctx context.Context, req *CBEngineBenchmarkRequestV2) (*CBEngineBenchmarkResponseV2, error)
```

When endpoints are upgraded, add new versioned types/functions alongside existing ones. Resources migrate at their own pace.

### Type naming

Request and response types include the version suffix and follow the pattern `<Domain><Entity><Purpose><Version>`:

```go
type DeviceGroupCreateRepresentationV1 struct { ... }
type CBEngineBenchmarkResponseV2 struct { ... }
```

## Schema Guidelines

- Keep schemas inline and as flat as possible.
- Favor nested attributes (`SingleNestedAttribute`, `SetNestedAttribute`, `ListNestedAttribute`) over blocks.
- **Terraform attribute names are always snake_case**, irrespective of how the upstream API formats the underlying JSON field. Translate at the boundary in input/state builders. Examples: API `categoryId` → TF `category_id`, API `osRequirements` → TF `os_requirements`, API `parameter4` → TF `parameter_4`. The only namespace exempt from this rule is **RSQL filter selectors** (`filters.FilterModel.Selector`), which pass through to the API verbatim and therefore retain their API-native spelling.

### Sets vs Lists

- **Sets** for user-supplied unordered collections where deduplication and order-independent comparison matter (e.g. `members`, `criteria`, `raw_component`).
- **Lists** for computed API results that are read-only. Sets require element hashing which adds overhead with no benefit when the user doesn't control the values.

Data source attributes returning API data should always use lists.

### Server-derived computed fields & `Optional+Computed` attributes

Pro endpoints commonly return **server-derived** values for fields the user did not set — a "no category" sentinel like `categoryId="-1"` / `categoryName="NONE"`, a default `priority="AFTER"`, etc. These show up in three places, each with its own pitfall, and all three must line up or Terraform errors with `Provider produced inconsistent result after apply`.

**1. Schema shape.** Any optional attribute the server fills in when omitted must be `Optional+Computed` so the framework knows the value can come from the server.

```go
"category_id": schema.StringAttribute{
    Optional: true,
    Computed: true,
    PlanModifiers: []planmodifier.String{
        stringplanmodifier.UseStateForUnknown(),
    },
},
```

A purely **read-only** server-derived field (e.g. `category_name`, derived from `category_id`) is `Computed`-only with the same `UseStateForUnknown` modifier — the user cannot set it, but its value carries across plans rather than going Unknown every refresh.

**2. Input builder must treat Unknown as nil.** When the user omits an Optional+Computed attribute, the plan value is **Unknown**, not Null. `types.String.ValueStringPointer()` returns a pointer to `""` for Unknown — Jamf Pro often rejects `categoryId: ""` with HTTP 500. Use the shared helper that nils both Null *and* Unknown:

```go
input := &pro.Script{
    Name:       plan.Name.ValueString(),
    CategoryID: helpers.OptionalStringPointer(plan.CategoryID),
    Priority:   helpers.OptionalStringPointer(plan.Priority),
    // ...
}
```

Available variants: `helpers.OptionalStringPointer`, `helpers.OptionalBoolPointer`, `helpers.OptionalInt64Pointer`. Apply uniformly to every Optional and Optional+Computed payload field. Do **not** call `types.String.ValueStringPointer()` directly on Optional+Computed attributes.

**3. Always refresh state via GET after Create AND Update.** Pro PUT responses are routinely **lossy** — `UpdateScriptV1` returns a `Script` without `categoryName` even though `GetScriptV1` does. If state is populated straight from the PUT response, server-derived fields go null while `UseStateForUnknown` carried the prior value into the plan — instant inconsistency error on Step 2 of any acceptance test that updates the resource.

```go
if _, err := r.client.UpdateScriptV1(updateCtx, plan.ID.ValueString(), buildScriptInput(plan)); err != nil {
    resp.Diagnostics.AddError("Error updating Jamf Pro script", err.Error())
    return
}

// PUT responses on Pro endpoints are lossy for server-derived fields
// (`categoryName` is omitted from PUT but present on GET). Refresh via GET
// so state is sourced from the canonical representation.
got, err := r.client.GetScriptV1(updateCtx, plan.ID.ValueString())
if err != nil {
    resp.Diagnostics.AddError("Error reading updated Jamf Pro script", err.Error())
    return
}
assignScriptResourceModel(&plan, got)
```

Create already follows this pattern (POST → HrefResponse → GET); Update must mirror it. Discard the PUT response's body — keep only the error.

**Reference implementation**: `internal/resources/pro/policies/script/` (resource, crud, input_builders, state_builders).

## Error Handling

Use the shared helpers from `internal/common/helpers` rather than rolling your own:

- `helpers.IsNotFoundError(err)` — 404 detection in `Read`/`Delete` operations.
- `helpers.IsServerError(err)` — 5xx detection for retry decisions.
- `helpers.ResolveTimeout(ctx, value, defaultDuration)` — resolve `framework-timeouts` values to a concrete deadline.
- `helpers.ReconcileOptionalBool` / `ReconcileOptionalInt` / `ReconcileOptionalString` / `Reconcile*Pointer` — preserve Terraform null/optional semantics when the API returns zero values.
- Wrap errors with `fmt.Errorf("context: %w", err)` to preserve the error chain.

## Naming Patterns

### Resources

Terraform construct names follow `jamfplatform_<domain>_<entity>` (or `jamfplatform_<domain>_<entity>_<verb>` for actions):

- `jamfplatform_device_group` (resource, data source, list resource)
- `jamfplatform_blueprints_blueprint` (resource, data source, list resource)
- `jamfplatform_cbengine_benchmark` (resource, data source, list resource)
- `jamfplatform_device_erase` / `_restart` / `_shutdown` / `_unmanage` (actions)

### Test names

Test functions use the pattern `TestAccResource_<Resource>_<Scenario>` for acceptance tests and `Test<Function>_<Case>` for unit tests:

```go
func TestAccResource_DeviceGroup_SmartComputer(t *testing.T) { ... }
func TestAccDataSource_Baselines(t *testing.T) { ... }
func TestBuildBlueprintInput_MinimalConfig(t *testing.T) { ... }
```

### Acceptance test resource names

Use the `tf-acc-` prefix for all resources created during acceptance tests:

```
tf-acc-static-computer
tf-acc-benchmark-all-rules
tf-acc-bp-scope-passcode
```

## Jamf Pro Resource Naming

Resources backed by the `pro/` or `proclassic/` packages of `jamfplatform-go-sdk` use a dedicated naming scheme that hides the API-layer split from users. Both layers expose Jamf Pro functionality; end users should not care which one is wired up.

### Rules

1. **Terraform construct name**: `jamfplatform_pro_<resource>` for every Jamf Pro resource, data source, list resource, or action — regardless of whether it is sourced from `pro/` or `proclassic/`.
2. **Slug source**: derive `<resource>` from the SDK filename, normalized to snake_case and singularized. Examples:
   - `pro/scripts.go` → `jamfplatform_pro_script`
   - `pro/categories.go` → `jamfplatform_pro_category`
   - `pro/smart_computer_groups.go` → `jamfplatform_pro_smart_computer_group`
   - `proclassic/networksegments.go` → `jamfplatform_pro_network_segment`
   - `proclassic/osxconfigurationprofiles.go` → `jamfplatform_pro_macos_configuration_profile` (override; modern terminology)
3. **Singular vs plural**:
   - Resource: singular (`pro_script`).
   - Data source: singular for ID/name lookup (`pro_script`), plural for filtered/list lookups (`pro_scripts`).
   - List resource: plural (`pro_scripts`).
   - Action: singular verb suffix (`pro_computer_erase`).
4. **Go package path**: `internal/resources/pro/<domain>/<resource>/` (two-tier). Domain groups related resources (e.g. `computers/`, `mobile_devices/`, `users/`, `policies/`, `configuration_profiles/`, `enrollment/`, `sso/`, `patch/`, `vpp/`, `settings/`, `inventory/`). Pick the closest fit; introduce a new domain folder if none applies. The leaf `<resource>` folder name is the **Terraform slug minus the `jamfplatform_pro_` prefix**, snake_case. Examples: `jamfplatform_pro_category` → `inventory/category/`; `jamfplatform_pro_self_service_plus_settings` → `settings/self_service_plus_settings/`; `jamfplatform_pro_smart_computer_group` → `computers/smart_computer_group/`. Do not drop descriptive suffixes (keep `_settings`, `_group`, etc.) — the folder name must match the Terraform slug exactly so future maintainers can grep one to find the other. The Go package declaration matches the folder name verbatim.
5. **Pro vs ProClassic preference**: default to `pro/`. Use `proclassic/` only when:
   - `pro/` has no equivalent endpoint, OR
   - `pro/` is materially less feature-complete (e.g. read-only when classic offers CRUD, missing required fields).
   - When both are wired across multiple resources, document the rationale in the resource's package-level comment.
6. **Overrides**: where the SDK filename is awkward, outdated, or ambiguous, override the Terraform slug. Record the override in `JAMF_PRO_INVENTORY.md` (gitignored planning file) at the time the batch is approved. There is no upfront override table — decisions happen per batch.
7. **Inventory tracking**: every Jamf Pro construct (planned, in-design, in-progress, shipped, skipped) is tracked in `JAMF_PRO_INVENTORY.md`. Not committed.

### Minimum Jamf Pro version check

The provider uses **one** `jamfplatform.Client` built from the standard `JAMFPLATFORM_*` credentials — there is no separate "Pro credentials" set. The same client serves Platform Services resources (`blueprints/*`, `cbengine/*`, `device_group`, etc.) and Jamf Pro resources (`pro/*`).

Pro resources gate themselves on a minimum Jamf Pro tenant version. Platform Services resources do not — they remain usable against tenants that don't have Jamf Pro provisioned.

Every Pro resource declares its minimum Jamf Pro version as an unexported `const` in `resource.go`:

```go
const minJamfProVersion = "11.5.0"  // empty string = no version check
```

- **Co-locate** with the resource. Do not centralize.
- **Empty string** (`""`) skips the check — use only when the resource genuinely works on the provider's declared minimum Pro version with no endpoint-specific floor.
- **Source the value** during the SDK-comparison phase of the resource-addition workflow ([CONTRIBUTING.md §Adding a Jamf Pro Resource](CONTRIBUTING.md#adding-a-jamf-pro-resource)). Record in `JAMF_PRO_INVENTORY.md`.

The tenant version is fetched **lazily** inside `providerdata.ConfigurePro` via `pd.GetJamfProVersion(ctx)`, which uses `sync.Once` to fetch via `GetJamfProVersionV1` exactly once per Data value and caches the result. Subsequent Pro Configure calls reuse the cached value.

#### `providerdata.Data` lives in its own package

The value Terraform passes to every Configure call (`req.ProviderData`) is `*providerdata.Data` — defined in `internal/providerdata/providerdata.go`, **not** `internal/provider/`. It carries the SDK client plus the `sync.Once`-cached lazy Pro version fetch.

- **Why a separate package**: `internal/provider/` already imports every resource package in order to register them. If resource packages also imported `internal/provider` to name the `Data` type, Go would reject the cycle. `internal/providerdata/` is a leaf package — resources import it, provider imports it, no loop.
- **Why a wrapper at all**: Platform Services resources only ever needed the raw `*jamfplatform.Client`, so the provider used to pass it directly as `ResourceData`. Pro resources require lazily-fetched, cached tenant Jamf Pro version state (so dozens of Pro resources in one config don't each re-call `GetJamfProVersionV1`). A `sync.Once` field on a shared struct is the natural shape, hence the wrapper.

#### Pro Configure: use the `providerdata.ConfigurePro` helper

Every Jamf Pro resource, data source, list resource and action funnels its Configure through `providerdata.ConfigurePro` — a single helper that performs the type assertion, fetches the cached tenant version, runs the per-resource version gate when set, and appends the provider-floor advisory warning when applicable. Do not hand-roll the boilerplate.

```go
const minJamfProVersion = "11.5.0"  // empty string = no version check

func (r *Resource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
    client, diags := providerdata.ConfigurePro(ctx, req.ProviderData, minJamfProVersion, "jamfplatform_pro_<name>")
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }
    r.client = client
}
```

The helper returns `(*pro.Client, diag.Diagnostics)`. When `req.ProviderData` is nil (early framework lifecycle) it returns `(nil, nil)` — leave `r.client` unset and let the next Configure call populate it. Data sources and list resources use the same one-liner shape; only the request/response types differ.

Platform Services resources do not use `ConfigurePro` — they only need the raw client. They type-assert `req.ProviderData.(*providerdata.Data)` and read `.Client` directly.

The tenant version is fetched **only when** a Pro construct with non-empty `minJamfProVersion` is in the config (or when the floor warning needs to run, which happens inside `ConfigurePro` after a `GetJamfProVersion` call). Configurations that use only Platform Services resources never trigger the fetch.

`RequireMinJamfProVersion` lives in `internal/common/helpers/pro_version.go` and:

- Tolerates Jamf's build-suffix format (`11.5.0-t1700000000` → parses as `11.5.0`).
- Parses `MAJOR.MINOR.PATCH`. Unparseable input → error diagnostic with the raw string.
- Empty `required` → no-op (returns nil diagnostics).
- `actual < required` → error diagnostic naming both versions and the resource type. **Error, not warning** — version-gated resources should not silently proceed against unsupported tenants.

#### Failure modes

| Condition | Outcome |
|-----------|---------|
| Tenant does not have Jamf Pro provisioned, Pro resource with non-empty `minJamfProVersion` used | `GetJamfProVersionV1` errors → propagated by `GetJamfProVersion` → resource Configure surfaces as "Failed to read Jamf Pro tenant version" |
| Tenant does not have Jamf Pro provisioned, Pro resource with empty `minJamfProVersion` used | Configure passes; first CRUD call surfaces the underlying Jamf API error |
| `GetJamfProVersionV1` returns a transient network/auth error | Same path as above; cached on `ProviderData` until next `terraform` invocation |
| Tenant version parses, `< required` | `RequireMinJamfProVersion` error |
| Tenant version unparseable | `RequireMinJamfProVersion` error with raw string |
| `minJamfProVersion = ""` | Version check skipped; no fetch triggered by this resource |

### Endpoint adoption & migration policy

**Scope: Jamf Pro resources only** (those backed by `jamfplatform-go-sdk/jamfplatform/pro/` or `proclassic/`). Pro APIs versionize endpoints (V1 / V2 / V3 / ...), can be deprecated by Jamf, and run on customer-provisioned Jamf Pro tenants with their own version drift. This policy governs when Pro resources move between endpoint versions. **All version transitions follow the same rules** — V1→V2, V2→V3, V3→V4, and so on. References to "V1" and "V2" below are generic placeholders for "current version N" and "new version N+1".

**Out of scope: Platform Services SDK packages** — `blueprints/`, `compliancebenchmarks/`, `devicegroups/`, `devices/`, `deviceactions/`, `ddmreport/`. These are continuously-deployed Jamf Platform microservices, not customer-versioned Jamf Pro endpoints. Always track the latest stable function in the SDK; no deprecation buffer, no quarterly audit, no annotation block required. When a Platform Services SDK function is updated (e.g., a new V2 added), migrate the corresponding resource opportunistically — typically as part of the SDK dependency bump that introduces it.

**ProClassic is unversioned**: `jamfplatform/proclassic/` functions do not carry V1/V2 suffixes. Migration timing (below) does not apply to ProClassic-backed resources. Annotation block is still recommended for documenting which SDK function is in use, but `Status:` simply tracks "current" against the SDK release in use.

#### SDK side-by-side dependency

The buffered migration timeline below assumes the SDK exposes both the deprecated version (N) and the new version (N+1) simultaneously during the deprecation window. **This applies to every version transition** — V1→V2, V2→V3, V3→V4, etc. — not just V1→V2.

As of writing, the SDK rarely keeps multiple versions: of ~565 versioned Pro endpoint bases, only 2 have side-by-side versions exposed. Generator change requested upstream — see [jamfplatform-go-sdk#19](https://github.com/Jamf-Concepts/jamfplatform-go-sdk/issues/19). Until that lands:

- When the SDK exposes both versions of an endpoint: follow the buffer policy below.
- When the SDK only exposes the new version on regeneration: migration is **SDK-bump-driven** — migrate at the SDK bump or pin the SDK to the prior release (only as a temporary workaround). Document the constraint in the resource's annotation block.
- When a Jamf deprecation is announced for an endpoint we use (at any version pair), file an issue against `jamfplatform-go-sdk` requesting retention of the deprecated version through our migration window.

#### Adoption (new endpoints)

| Scenario | Action |
|----------|--------|
| Building a brand-new resource, SDK exposes V1 only | Use V1. |
| Building a brand-new resource, SDK exposes V2 (V1 still current) | **Use V2.** Default to latest stable on new resources — no migration cost. |
| Building a brand-new resource, SDK exposes V2 marked beta/EAP | Use V1. V2 only if V1 is unworkable; document the choice in the resource. |
| Existing **shipped** resource on V1, new V2 lands (V1 not deprecated) | **Do not migrate.** Wait for deprecation, fast-track trigger, or material feature gap. |
| Existing shipped resource on V1, V2 only adds optional new fields | Migrate opportunistically, e.g., when next touching the resource. Not urgent. |
| Existing shipped resource on V1, V2 fixes an issue meeting fast-track criteria | **Fast-track migrate** — see below. |

Rationale: migrating shipped resources costs schema/state-upgrader churn. Don't pay it without reason. New resources have no migration cost.

#### Deprecation migration timeline

Jamf's typical window: ~12 months from deprecation announcement to removal.

| Trigger | Action |
|---------|--------|
| Jamf marks V1 deprecated | Open a migration tracking issue within **30 days**. Schedule the work. |
| **6 months** after deprecation announcement (soft default) | Migration **should be merged**. Soft target. |
| **3 months before announced removal date** (hard floor) | Migration **must be merged**. Hard floor — no shipped resource may sit on a deprecated endpoint inside this window. If migration is not yet shipped, it becomes top priority and bumps the buildout roadmap. |
| Removal date | All migrations must already be shipped. Anything not migrated is a regression. |

The 6-month soft default and 3-month hard floor work as a pair: 6-month catches the common case, 3-month catches edge cases (Jamf shortens the window, a migration slips, a deprecation announcement was missed).

#### Fast-track exception

Skip the 6-month buffer and migrate as soon as feasible when V2 includes one of:

- **Security fix** (auth, scope, data exposure).
- **Data integrity** issue (V1 returns wrong values, silent truncation, race conditions).
- **Breaking bug** affecting a documented common use case where no workaround exists.
- **Critical feature** required by a user-reported issue with no viable workaround.

Maintainer call, documented in the migration PR description.

#### Slow-track exception

When migration would require a **breaking Terraform schema change**, defer to the next provider major version. Document explicitly in the resource's `crud.go` annotation (see below) and in the major-version release planning.

#### Tracking — in-code annotation (Pro resources only)

Every Jamf Pro resource's `crud.go` carries an annotation block at the top of the file listing the SDK endpoints in use, their status, and the last maintainer review date. Platform Services resources do **not** carry this annotation.

```go
// SDK endpoints used:
//   pro.CreateScriptV2 (introduced Jamf Pro 11.8)
//   pro.ReadScriptV2   (introduced Jamf Pro 11.8)
//   pro.UpdateScriptV2 (introduced Jamf Pro 11.8)
//   pro.DeleteScriptV2 (introduced Jamf Pro 11.8)
//
// Status: current. Last reviewed 2026-05-18.
```

When V1 becomes deprecated, change the line:

```go
// Status: deprecated by Jamf YYYY-MM-DD; migrate by YYYY-MM-DD (6mo soft / 3mo hard floor). Last reviewed YYYY-MM-DD.
```

When the migration is in flight:

```go
// Status: migration in progress on V3 in PR #NNN. Last reviewed YYYY-MM-DD.
```

This annotation is the single source of truth. No separate `ENDPOINT_VERSIONS.md` to keep in sync.

#### Quarterly audit cadence (Pro resources only)

Once per quarter, a maintainer:

1. Scans Jamf Pro release notes and the `jamfplatform-go-sdk` changelog for new Pro endpoint versions and deprecations (Pro / ProClassic only — Platform Services microservices are out of scope).
2. Greps the repo for endpoint annotations: `grep -rA5 "SDK endpoints used:" internal/resources/pro/`.
3. Reconciles each annotation against current Jamf + SDK state.
4. Updates `Last reviewed YYYY-MM-DD` on every annotation that's still current.
5. Opens migration tracking issues for any newly-deprecated Pro endpoints.
6. Flags any annotation whose `Last reviewed` date is older than 120 days (means a prior audit was skipped).

The `Last reviewed` date is the drift detector — any annotation more than a quarter stale signals the audit hasn't happened.

### ID type handling

Jamf endpoints expose mixed ID shapes: `pro/` uses integer IDs for many resources and UUIDs for newer ones; `proclassic/` is almost exclusively integer IDs. Terraform's `ID` attribute is always `types.String`.

**Convention**: always stringify integer IDs in Terraform state. Convert back to `int` / `int64` when calling the SDK.

Helpers (to be added under `internal/common/helpers/ids.go` when the first Pro resource lands):

- `IntIDToString(id int64) types.String`
- `StringToIntID(s types.String) (int64, error)`
- UUIDs pass through unchanged as `types.String`.

`ImportState` parses the imported string per the rules below before populating state.

### Import format

Default convention for `ImportState`:

- **Single ID**: `terraform import jamfplatform_pro_<name>.foo <id>` — where `<id>` is the resource's primary key (integer or UUID as Jamf exposes it). Standard `resource.ImportStatePassthroughID` usage in most cases.
- **Composite / parent-scoped** (rare): `terraform import jamfplatform_pro_<name>.foo <parent_id>:<id>` — colon-separated. Parse with `strings.SplitN(req.ID, ":", 2)`. Validate both segments before setting state. Document the expected format in the resource's `MarkdownDescription` and in `examples/resources/<name>/import.sh`.

Avoid name-based imports; use IDs only.

### Endpoint shape classification

During the SDK-comparison gate ([CONTRIBUTING.md §Adding a Jamf Pro Resource](CONTRIBUTING.md#adding-a-jamf-pro-resource) step 3), classify the endpoint based on which operations the SDK exposes:

| Operations available | Construct type | Notes |
|----------------------|----------------|-------|
| Create + Read + Update + Delete | `resource` (+ usually a singular `data source` and a plural `list resource`) | Standard CRUD |
| Read only (single + list, or list only) | `data source` (+ plural `data source` or `list resource`) | No state-managed object |
| Update only (no Create/Delete) | `resource` flagged as **singleton** — one record per tenant | See [Singleton resources](#singleton-resources) below for the full convention (fixed ID, Create→Update, no-op Delete, import format). E.g., `activation_code`, `client_check_in`, `jamf_pro_server_url`, `self_service_plus_settings`. |
| Fire-and-forget command (Create returns command ID, no Read/Update/Delete) | `action` | E.g., `pro_computer_erase`, `pro_computer_restart` |

Record the classification in `JAMF_PRO_INVENTORY.md` Notes column during the in-design phase.

### Singleton resources

Jamf Pro objects that exist one-per-tenant and are exposed as Update-only on the API are modeled as **singleton** resources. The whole convention below is the load-bearing definition — any new singleton must follow it.

**Domain folder**: `internal/resources/pro/settings/<resource>/`. The `settings/` domain is the canonical home for Pro singletons (`activation_code`, `client_check_in`, `self_service_plus_settings`, `jamf_pro_*`). Reference template: `internal/resources/pro/settings/self_service_plus_settings/`.

**Leaf folder name**: matches the Terraform slug exactly (rule #4 above), e.g. `self_service_plus_settings/` for `jamfplatform_pro_self_service_plus_settings`.

**Fixed ID**: every singleton stores `helpers.SingletonID` (`"singleton"`, defined in `internal/common/helpers/singleton.go`) as its Terraform state ID. Set it in Create, Read, and Update via `state.ID = types.StringValue(helpers.SingletonID)`. Do not import any other identifier.

**Identity schema**: declare a single `id` string attribute with `RequiredForImport: true` — same shape as CRUD resources, just always populated with `helpers.SingletonID`.

**Schema `id` attribute**: `Computed: true` with `stringplanmodifier.UseStateForUnknown()`. Never `Required` (users cannot pick the ID — it is always `"singleton"`).

**Create**: the Jamf Pro API has no Create endpoint for singletons (the record already exists). Funnel `Create` into the same `Update<X>V1` SDK call used by `Update`, then `Get<X>V1` to capture authoritative state, set `state.ID = types.StringValue(helpers.SingletonID)`. The post-write `Get` is mandatory — it picks up any server-side transformations, computed defaults, or future field additions without requiring code changes.

**Delete**: no-op on the remote — the record cannot be deleted. The handler signature uses `_` markers on the unused `req` / `resp` parameters to signal the omission is intentional, and emits a single `tflog.Trace` line. No state mutation; Terraform removes the resource from state on its own after the handler returns. Document this in the handler doc-comment so reviewers understand the omitted SDK call is deliberate.

**Defensive nil-client guard**: every CRUD handler (`Create`, `Read`, `Update`) opens with `if r.client == nil { resp.Diagnostics.AddError(providerNotConfiguredError()) ; return }` before reading plan/state. Defense-in-depth against framework lifecycle edge cases or misconfigured provider blocks routing to CRUD with an unconfigured client. Centralize the message in a package-local helper so the wording stays uniform across the three handlers. The `Delete` no-op does not need the guard — it makes no SDK call.

**Import**: `terraform import <type>.<name> singleton`. `ImportState` **must** validate `req.ID == helpers.SingletonID` and reject any other identifier with a clear error diagnostic — silent normalization on the next Read masks the mis-import and confuses users. Pass the validated ID through to `resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)`. The example `import.sh` under `examples/resources/<type>/import.sh` must contain this exact command.

**No list resource, no plural data source**: singletons have nothing to list. Provide the singular data source only when reading the current settings without managing them is useful.

**State builders — nil-fallback semantics**: SDK structs for singleton fields commonly use pointer types (`*bool`, `*string`) with `json:",omitempty"`. When the API response is nil for a **Required** schema attribute, fall back to the field's zero value with an explicit doc-comment explaining why (Required attributes cannot be null in committed state). When the field is **Optional**, prefer `helpers.ReconcileOptionalBoolPointer` / `ReconcileOptionalStringPointer` so user-set nulls are preserved rather than collapsed. The assigner **must not** write `state.ID` — the CRUD handler is responsible for stamping `helpers.SingletonID` after the assign call, and a test should pin that the assigner leaves any pre-existing ID untouched.

**Acceptance test**: `CheckDestroy` is wired but inverted — it asserts the record **still exists** on the tenant after Terraform destroys the resource from state (the singleton-specific shape of the standard CheckDestroy contract). Test the two Update paths (e.g. toggle a bool true→false), import with `ImportStateId: "singleton"`, and a dedicated step asserting `ImportStateId: "not-the-singleton"` is rejected with `ExpectError` matching the ImportState error summary.

**Unit tests**: a `state_builders_test.go` is mandatory — it pins (a) nil and non-nil round-trip for every assigner, (b) that the assigner does not clobber `state.ID`, and (c) that `helpers.SingletonID` is `"singleton"` (catches accidental drift in the constant).

**Before opening the PR**: run `make fix fmt lint test` (must be clean) then `make generate` to rebuild `docs/resources/pro_<name>.md` and `docs/data-sources/pro_<name>.md`. Commit the generated docs with the source.

### Pro error/retry helpers (planned extension)

Existing `internal/common/helpers/helpers.go` provides `IsNotFoundError` and `IsServerError`. Pro APIs add cases the existing helpers don't cover:

- **429 Too Many Requests** — Pro endpoints rate-limit aggressively under load.
- **423 Locked** — Pro async operations (deploy, sync, redeploy) return `423` while another operation is in flight.
- **409 Conflict** — Pro PATCH/PUT on stale state.

When **3 or more** Pro resources need retry handling for these cases, extract into `internal/common/helpers`:

- `IsRateLimitError(err error) bool` — 429
- `IsLockedError(err error) bool` — 423
- `IsConflictError(err error) bool` — 409
- `RetryWithBackoff(ctx, op, isRetriable, maxAttempts) error` — generic retry helper that respects context deadlines.

Until the trigger fires, retry logic lives in-resource. Same deferred-abstraction discipline as shared schemas.

### Provider overall minimum Jamf Pro version (advisory warning)

Independent of per-resource `minJamfProVersion` constants (which are hard errors), the provider declares an **overall recommended minimum Jamf Pro version**: the highest version any shipped Pro resource requires. Surfaces as a **warning, not an error**, when the tenant version is below this floor:

```go
// internal/providerdata/providerdata.go
const ProviderMinJamfProVersion = "11.27.0"  // bump at release time
```

Every Pro resource funnels its Configure through `providerdata.ConfigurePro`, which calls `pd.GetJamfProVersion(ctx)` unconditionally — regardless of whether the resource declares a per-resource `minJamfProVersion` const. The call is cached via `sync.Once` on `providerdata.Data`, so it fires at most once per `terraform` invocation. After the version is cached, the helper computes the floor warning and appends it to the Configure response.

When the fetch errors (e.g. 403 on a non-Jamf-Pro tenant): the helper swallows the error silently for resources with empty `minJamfProVersion` (the SDK CRUD call will surface the real failure later); for resources with a non-empty `minJamfProVersion` the helper surfaces the fetch error as a Configure-time hard error ("Failed to read Jamf Pro tenant version …"). Always-fetch + selective-swallow is the shape — that is exactly what `ConfigurePro` does, so callers do not need to think about it.

The warning text:

> Jamf Pro tenant older than provider build target
>
> This provider release was built against the Jamf Pro API as of version `A.B.C`. The tenant reports `X.Y.Z`. Some Pro resources may rely on endpoints or fields that did not exist in the tenant's version and could fail at apply time. Upgrade Jamf Pro or pin an older provider release.

Rationale: per-resource `const minJamfProVersion` covers hard correctness for individual resources that need newer endpoints. The provider-level floor catches the broader case where the user's tenant lags the API spec the provider was generated from — many Pro resources may quietly rely on fields or endpoints that only exist in newer Jamf Pro versions without each declaring a per-resource gate. Warning is enough — the per-resource error remains the safety net for resources that have one.

**Release-time process**: before tagging a release, grep all `minJamfProVersion` constants under `internal/resources/pro/`, take the max, update `ProviderMinJamfProVersion` in `internal/providerdata/providerdata.go` if it has moved. The provider Schema description interpolates the const so `docs/index.md` updates automatically after `make generate`. The hard-coded version string in `README.md` (under "Supported Jamf products and tenant version targets") must be bumped by hand. When additional Jamf products are added in the future, each gets its own `Provider<Product>MinVersion` constant + row in the provider Schema table + row in the README table; the release-time process expands accordingly.

### Shared schemas (deferred abstraction)

Many Jamf Pro resources expose similar shapes (scope, site, category, criteria, self-service payload). **Do not extract these into shared schemas upfront** — superficially similar Jamf APIs often differ in field names, ID types (int vs UUID string), and null semantics. Premature abstraction here produces helpers with per-resource branching that is harder to read than the original duplication.

**Refactor trigger**: when 3 or more shipped resources have a verified-identical shape (same fields, same types, same null semantics — checked against the SDK structs, not eyeballed), extract a helper under `internal/common/schemas/`. Not before.
