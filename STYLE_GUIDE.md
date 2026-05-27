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
- `howett.net/plist` (BSD-2-Clause) — Apple plist parser/serialiser. Required by `internal/resources/pro/configuration_profiles/macos_configuration_profile/` to compare user-supplied `.mobileconfig` payloads against the server-canonical form for diff suppression. Use is contained to the configuration-profile resource family; no other code path should import it.

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

### Attribute names mirror the Jamf Pro admin UI when the wire name is cryptic

Jamf Pro's wire payloads frequently use short, internal names that do not match what administrators see in the admin UI — `cache_last_user` is labelled "Create Mobile Account" in the AD binding screen; `mount_style` is labelled "Network Protocol"; `uid` is labelled "Map UID to attribute"; and so on. A TF user reading the provider docs should not have to translate from the wire spelling back to the UI label they know.

**Rule**: when the wire name is cryptic or differs materially from the admin-UI label, **rename the Terraform attribute to mirror the UI label** (snake_case, of course). Translate at the boundary in input/state builders — the wire name stays inside the SDK call. Examples from `jamfplatform_pro_directory_binding`:

| UI label | TF attribute | Wire element |
|----------|--------------|--------------|
| Create Mobile Account | `create_mobile_account` | `cache_last_user` |
| Network Protocol | `network_protocol` | `mount_style` |
| Map UID to attribute | `uid_attribute_mapping` | `uid` |
| Force local home directory on startup disk | `force_local_home_directory` | `local_home` (bool, AD type) |
| Home Location | `home_location` | `local_home` (string, ADmitMac type) |
| Allow administration by | `admin_group` / `admin_groups` | `admin_group` / `admin_groups` |

When the wire name is already a reasonable match for the UI label (e.g. `encrypt_using_ssl`, `use_unc_path`, `workstation_mode`), leave it alone — gratuitous renames produce churn without payoff.

**Every renamed attribute's `MarkdownDescription`** must lead with the exact UI label in bold quoted form (e.g. `**"Create Mobile Account"** in the Jamf Pro admin UI.`) so a `terraform plan` reviewer can match against the admin screen. The wire name should **not** appear in user-facing text — record it instead in a Go comment immediately above the attribute, where future maintainers searching for the wire name can find it without exposing API plumbing to end users.

```go
// Wire element: cache_last_user — renamed for UI alignment.
"create_mobile_account": optBool("**\"Create Mobile Account\"** in the Jamf Pro admin UI. …"),
```

The rename rule applies to all current and future Jamf Pro resources. Where an existing shipped resource has cryptic wire-name attributes, do not retrofit in a feature PR — schedule a dedicated rename PR (the change is breaking for users).

### User-facing descriptions are UI-aligned, not wire-aligned

`MarkdownDescription` / `Description` strings on every `pro_` schema attribute, resource, data source, and list resource are **user-facing Terraform Registry documentation**. They render in the registry and in the IDE schema browser, side-by-side with the Jamf Pro admin UI. The provider exists to abstract Jamf Pro's API plumbing — descriptions should reflect that abstraction, not betray it.

**Strip from user-facing descriptions** (move to Go comments above the attribute if the maintainer-side info is still useful):

- `<xml_tag>` references and any other wire-shape literals (e.g. `<level>`, `<jss_users>`, `<security><password>`).
- "Wire field …" / "wire field …" / "wire emits …" / "wire-symmetric" / "on the wire" / "wire shape" / "wire-canonical" framing.
- "classic API" / "classic endpoint" mentions, and endpoint paths (`/api/proclassic/...`, `/api/pro/v1/...`, `/JSSResource/...`).
- HTTP method names (`POST`, `PUT`, `GET`, `DELETE`) — rephrase as `create`/`update`/`read`/`delete` if the action matters to the user.
- SDK package names, SDK function names, Go type names, internal helper names.
- "Server-derived" — replace with `Returned by Jamf Pro; not user-settable` or `Read-only`.

**Keep — but rephrase without wire jargon**:

- Write-only / read-back quirks (e.g. *"Jamf Pro does not echo this value back after it is set, so the provider treats it as write-only."*).
- Field conflicts and validator notes (*"conflicts with X if Y"*).
- Special-case behaviour (`-1 means none`, `0 disables`, valid value lists).
- Cross-field relationships (*"Pair with X to set Y"*).
- Mode-dependent or singleton-resource notes (e.g. SSO OIDC vs SAML, tenant-wide singletons).

**Endpoint references and SDK annotations** belong in maintainer-side surfaces only:

- The `crud.go` `// SDK endpoints:` annotation block at the top of every Pro/ProClassic resource (see `#### Tracking — in-code annotation` below).
- Go file/function comments.
- Never in `MarkdownDescription` strings.

The reference implementations for the rewritten tone are `internal/resources/pro/configuration_profiles/macos_configuration_profile/resource.go` and `internal/resources/pro/configuration_profiles/mobile_device_configuration_profile/resource.go` — copy that voice for new resources.

### Sets vs Lists

- **Sets** for user-supplied unordered collections where deduplication and order-independent comparison matter (e.g. `members`, `criteria`, `raw_component`).
- **Lists** for computed API results that are read-only. Sets require element hashing which adds overhead with no benefit when the user doesn't control the values.

Data source attributes returning API data should always use lists.

### Plaintext secrets — `WriteOnly` with `_wo_version` rotation companion

New Pro resources exposing a user-supplied plaintext secret (passwords, API tokens, shared keys) **MUST** model it as `Optional + Sensitive + WriteOnly`. The plaintext is sent to Jamf on writes but never persisted in Terraform state — the framework strips it. Storing plaintext in state leaks credentials to anyone with state-file read access; "we mark it Sensitive in the schema" is not enough — Sensitive only redacts CLI output, the raw plaintext still lives in the state file.

Every WriteOnly secret **MUST** carry a sibling `<attr>_wo_version` rotation trigger:

```go
"password": schema.StringAttribute{
    MarkdownDescription: "...",
    Optional:            true,
    Sensitive:           true,
    WriteOnly:           true,
},
"password_wo_version": schema.Int64Attribute{
    MarkdownDescription: "Rotation trigger for the `WriteOnly` `password`. Bump to force a re-PUT.",
    Optional:            true,
},
```

Why: WriteOnly values are excluded from Terraform's drift detection, so the user has no way to force a re-PUT by changing the password value alone — Terraform sees no diff. Bumping the Int64 companion is the only signal the provider has that "the user wants to rotate". Without it, the only way to rotate a stored password is `terraform destroy` + recreate. Pattern matches HashiCorp's documented `_wo_version` convention.

CRUD wiring:

- Create + Update **MUST** call `req.Config.Get(ctx, &cfg)` — the WriteOnly plaintext lives in `cfg`, not `plan` (the framework nullifies WriteOnly attrs in `plan`).
- Update **MUST** also call `req.State.Get(ctx, &state)` to compare `plan.<attr>_wo_version` against `state.<attr>_wo_version`; include the plaintext on the wire only when they differ (or thread it unconditionally if the resource's server semantics require the field on every write — e.g. `jamfplatform_pro_policy.account_maintenance`, where omitting the password erases it server-side and breaks the next client run).
- State builders **MUST NOT** preserve the plaintext across reads — the framework strips it from state regardless. The `_wo_version` companion is a regular Optional Int64 and **MUST** round-trip from prior state (the wire never echoes it).

`*_sha256` and similar server-redaction sentinels are **forbidden**:

- The classic API returns the literal string `********************` (20 asterisks) regardless of stored password content — it carries no drift-detection signal.
- Surfacing it as a Computed sibling encourages users to treat it as a real hash and write false-positive drift assertions.
- New Pro resources MUST NOT add a `*_sha256` Computed attribute alongside a `WriteOnly` plaintext.

`SetNestedAttribute` cannot contain `WriteOnly` children — the framework refuses to load the schema. If the wire shape carries a Set of nested objects with a plaintext secret (e.g. `account_maintenance.accounts[].password` on the classic policy), the surrounding attribute **MUST** be a `ListNestedAttribute`. Reorder wire entries by a stable natural key (username, name) when flattening so positional identity round-trips through unordered server responses — see `internal/resources/pro/policies/policy/state_builders.go` `flattenPolicyAccountMaintenance` for the reference pattern.

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

**1a. Nested-list elements: use `UseNonNullStateForUnknown`.** `Optional+Computed` scalars inside a `ListNestedAttribute` or `SetNestedAttribute` MUST use `stringplanmodifier.UseNonNullStateForUnknown()` (and bool/int siblings) rather than `UseStateForUnknown`. `UseStateForUnknown` copies the prior `StateValue` into the plan — including `Null` — and for an appended list element the prior state at the new index is `Null`. When the server then returns a value for that field on the new element, the framework consistency check trips with `Provider produced inconsistent result after apply`. If a `Sensitive` sibling (e.g. a `WriteOnly` password) lives on the same nested element, the error path is redacted up to the nearest non-sensitive ancestor, masking the real attribute and producing the misleading `.<parent>: inconsistent values for sensitive attribute`. `UseNonNullStateForUnknown` skips the copy when prior state is `Null`, leaving the plan `Unknown` so any post-apply value is accepted. Behavior is identical for the non-Null case (singletons, already-set values), so prefer this modifier uniformly within nested-collection element schemas. Reference: `internal/resources/pro/policies/policy/resource.go` `optComputedString` / `optComputedBool` / `optComputedInt` helpers.

**2. Input builder must treat Unknown as nil.** When the user omits an Optional+Computed attribute, the plan value is **Unknown**, not Null. `types.String.ValueStringPointer()` returns a pointer to `""` for Unknown — Jamf Pro often rejects `categoryId: ""` with HTTP 500. Use the shared helper that nils both Null *and* Unknown:

```go
input := &pro.Script{
    Name:       plan.Name.ValueString(),
    CategoryID: helpers.OptionalStringPointer(plan.CategoryID),
    Priority:   helpers.OptionalStringPointer(plan.Priority),
    // ...
}
```

Apply `helpers.OptionalStringPointer` uniformly to every Optional and Optional+Computed string payload field. Do **not** call `types.String.ValueStringPointer()` directly on Optional+Computed attributes. Add bool/int variants only when a resource actually needs them — match the SDK pointer type at the helper signature so call sites don't need casts.

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

**4. State builders must use `Reconcile*Pointer` for every Optional and Optional+Computed string field.** A bare `helpers.StringPointerValueOrNull(s.X)` overwrites Terraform's null/empty distinction every refresh; `helpers.ReconcileOptionalStringPointer(s.X, state.X)` preserves the user's prior value (including an explicit empty string) when the API returns nothing. Apply uniformly — do not mix `StringPointerValueOrNull` and `ReconcileOptionalStringPointer` across the same model. Only fully `Computed`-only fields (no `Optional`) should use `StringPointerValueOrNull` because there is no user value to reconcile.

**Reference implementation**: `internal/resources/pro/policies/script/` (resource, crud, input_builders, state_builders).

**4a. Use `helpers.PreserveStringWhenWireEmpty` for user-authored strings the Classic API may echo as empty.** Some Jamf Pro Classic endpoints (observed on configuration-profile self-service blocks — `self_service_description`, `notification_subject`, `notification_message`, `authorization_password`) return the field as `<elem></elem>` on read even after a successful write of a non-empty value. `ReconcileOptionalStringPointer` treats an empty echo as "user did not set it" and collapses state to Null. When the plan held a non-empty user-authored string, the Null state then trips Terraform Core's `"Provider produced inconsistent result after apply"` check. **Watch out**: when the affected attribute lives in a block that *also* contains a `Sensitive` sibling, Terraform Core masks the error path to the parent block (e.g. `.self_service: inconsistent values for sensitive attribute` instead of `.self_service.self_service_description: ...`) — the wording falsely implicates the Sensitive attribute and sends debugging in the wrong direction. Default rule: any user-authored string field under a block that carries a Sensitive sibling should use `PreserveStringWhenWireEmpty` rather than `ReconcileOptionalStringPointer`. Sentinel-test for this pattern with a unit test exercising a non-empty configured value plus an empty-string wire echo.

### `SingleNestedAttribute` blocks: Optional-only when the model uses typed-pointer

Nested blocks modelled as `*StructModel` (typed pointer to a struct with `tfsdk:` tags on every field) cannot be `Optional+Computed`. The Plugin Framework decodes an absent-but-Computed block as **Unknown**, and `*StructModel` has no representation for Unknown — apply fails with:

> `Received unknown value, however the target type cannot handle unknown values. Use the corresponding `types` package type or a custom type that handles unknown values.`

Two ways out:

1. **Keep the block `Optional`-only.** Inner fields can still be `Optional+Computed` so the server may populate defaults the user omitted. This is the right default — typed-pointer models are easier to read and write than `types.Object`-shaped ones. Document for users: supply the block (even empty: `<type>_block = {}`) to take management of the per-type configuration. Omitting the block entirely for a record whose server-side representation includes a populated block will produce drift on the next refresh.
2. **Switch the model field to `types.Object`** with an `attrTypes` map. Only worth it when the resource genuinely needs the framework to represent the block as Unknown — most don't.

Reference: `internal/resources/pro/inventory/directory_binding/` — five per-type nested blocks, all Optional-only, inner fields Optional+Computed.

### Asymmetric server normalisation on `type`-style discriminator fields

Some classic endpoints accept a **legacy product name** on write but normalise to the **modern product name** on read. Example: PowerBroker directory bindings — the `/directorybindings` create path rejects `type="PowerBroker Identity Services"` with HTTP 409 "Problem with directory binding type" and only accepts `type="Likewise"` (the pre-acquisition name); the read path always returns `type="PowerBroker Identity Services"`. Pass the alias through to TF state and users get gratuitous drift; pass the modern name through to the server and Create silently fails.

Pattern: translate one-way at the input boundary so the wire-canonical name is what users see in state, and the legacy alias is an implementation detail. Centralise the mapping in `helpers.go`:

```go
const typePowerBrokerCreateAlias = "Likewise"

func mapType(tfType string) string {
    if tfType == typePowerBroker {
        return typePowerBrokerCreateAlias
    }
    return tfType
}
```

Inside `input_builders.go`, route the `type` field through the mapper before emitting the SDK payload. Cover the mapping with a unit test (input X → wire Y) so future maintainers don't quietly delete it.

Reference: `internal/resources/pro/inventory/directory_binding/helpers.go` (`mapType` + `typePowerBrokerCreateAlias`).

#### Plan-modifier rewrites are NOT a valid option for `Required` attributes

The "translate at the input boundary" pattern above only works because the user-facing TF value and the wire value are **identical** in directory_binding — the alias is swapped inside the input builder, not in the schema. If you instead try to canonicalise via a `planmodifier.String` that rewrites `req.ConfigValue` (e.g. lowercasing `Individual And Institutional` → `Individual and Institutional` at plan time), Terraform's plugin framework rejects the apply with:

```
Error: Provider produced invalid plan

Provider planned an invalid value for <attribute>:
planned value cty.StringVal("…") does not match config value cty.StringVal("…").

This is a bug in the provider, which should be reported in the provider's own issue tracker.
```

The framework enforces `plan == config` for any attribute that is **not** `Computed`. Required attributes can't be rewritten by plan modifiers. The two valid options are:

1. **Strict wire-form acceptance** — schema validator (`stringvalidator.OneOf`) accepts only the canonical wire spellings; user must supply them verbatim. Drift surfaces at plan time as a validator error, not at apply time as a framework violation. **Prefer this** for simple discriminator fields.
2. **Optional + Computed with plan-modifier rewrite** — works mechanically, but turns the attribute into a "computed override" and complicates required-input semantics (the user can omit it). Only worth it when the canonicalisation is more than a case-fold (e.g. an alias swap that genuinely round-trips).

The mistake to avoid: pairing `Required: true` with `OneOfCaseInsensitive` + a canonicalising plan modifier. The validator passes, but the framework will reject the plan at apply time. Caught during `disk_encryption_configuration` build, 2026-05-23.

References: `internal/resources/pro/inventory/directory_binding/resource.go` (Required + OneOf, no plan modifier — preferred pattern); `internal/resources/pro/inventory/disk_encryption_configuration/resource.go` (`key_type` accepts only `Individual and Institutional` with lowercase `and`).

### Working around server-broken classic endpoints

Some Jamf Pro classic endpoints are genuinely broken — the `/directorybindings/name/{name}` endpoint returns HTTP 500 for every name lookup as of 2026-05-23, even when the binding exists and is reachable by ID. List + ID paths work; only name lookup is dead.

When the SDK call backed by a broken endpoint is on a non-critical path (data source name resolver, optional fallback), route around it inside the resource package rather than waiting on the upstream fix:

- Implement a private helper in the resource package (e.g. `lookupByName` in `data_source.go`) that does **List + client-side name match + GetByID**.
- Doc-comment the helper with the bug summary, the date observed, and the upstream fix that would let the helper be deleted.
- Open an SDK issue so every consumer benefits once the SDK adopts the fallback (or once the server is fixed).

Reference: `internal/resources/pro/inventory/directory_binding/data_source.go` (`lookupByName`).

### Cross-field validation

**Required pattern:** every cross-field rule MUST be expressed as an **attribute-level validator** (`Validators: []validator.Bool{...}` on the schema attribute itself), not as a resource-level `resource.ResourceWithConfigValidators`. Errors attach to the offending attribute, the rule is co-located with the schema, the diagnostic is field-named, and the convention matches every resource in the provider.

**Decision matrix — which validator to reach for:**

| Rule shape | Use |
|---|---|
| "A and B are paired — supplying either requires the other" (any value triggers) | Off-the-shelf `boolvalidator.AlsoRequires` / `stringvalidator.AlsoRequires` / `int64validator.AlsoRequires` |
| "A and B are mutually exclusive" (any value triggers) | Off-the-shelf `boolvalidator.ConflictsWith` / `stringvalidator.ConflictsWith` / `int64validator.ConflictsWith` |
| "Enum membership" | Off-the-shelf `stringvalidator.OneOf(values...)` |
| "Required only when this bool is `true`" or "required only when this string equals `\"X\"`" — **value-specific** | Custom `validator.Bool` / `validator.String` in the package's `validators.go` |
| "Exactly one of X / Y / Z" spanning several siblings | Framework `resourcevalidator.ExactlyOneOf` / `AtLeastOneOf` / `Conflicting` |

**Rule of thumb:** the off-the-shelf `AlsoRequires` / `ConflictsWith` fire whenever the validated attribute is *known* — true OR false, `"foo"` OR `""`. They do NOT inspect the value. **If the rule only applies for a specific value, the off-the-shelf semantics are wrong and a custom validator is the right tool — but only then.** Writing a custom validator when an off-the-shelf one fits is a code-review reject.

**Path syntax:** `path.MatchRoot("other_attr")` for a root-level sibling; `path.MatchRelative().AtParent().AtName("other_attr")` for a sibling under the same nested object.

**Off-the-shelf references:**
- `internal/resources/blueprints/blueprint/components/math_settings.go` — paired toggles with `boolvalidator.AlsoRequires`.
- `internal/resources/blueprints/blueprint/components/software_update.go` — mix of `AlsoRequires`, `ConflictsWith`, `RegexMatches`, `Between`.
- `internal/resources/pro/inventory/network_segment/resource.go` — `override_buildings` / `override_departments` pair with their companion string via `AlsoRequires`.
- `internal/actions/device/erase.go` — toggle + companion via `AlsoRequires`.

**Custom validator references (value-specific only):**
- `internal/common/scope/validators.go` — `AllFlagConflictsWith` is a `validator.Bool` that fires only when the bool is `true`; off-the-shelf `ConflictsWith` would incorrectly fire when the bool is `false` too.
- `internal/resources/pro/settings/sso_settings/validators.go` — value-discriminated validators for `configuration_type ∈ {SAML, OIDC, OIDC_WITH_SAML}`, `metadata_source ∈ {URL, FILE}`, `setup_type = "UPLOADED"`, etc., each requiring different companion sets.

**Custom validator authoring rules:**
- Implement `Validate{Bool,String,Int64,...}` on a struct in `validators.go`.
- Skip when `req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown()` — apply-time references must not false-positive.
- Skip when the value does not match the discriminator (e.g. `req.ConfigValue.ValueBool() == false` for a "true-only" rule).
- Read companions via `req.Config.GetAttribute(ctx, path.Root("…") / path.MatchRelative()..., &target)`.
- Attach errors to the **companion's** path (not the discriminator's), via `resp.Diagnostics.AddAttributeError(companionPath, summary, detail)` — that's where the user needs to look.

Avoid `resource.ResourceWithConfigValidators` even for multi-attribute rules unless the framework's `resourcevalidator` package genuinely cannot express them. Bespoke whole-config validators are a last resort.

### Configuration profile payload diff suppression (mask-and-compare)

Jamf classic configuration-profile endpoints (`osxconfigurationprofiles`, `mobiledeviceconfigurationprofiles`) re-serialise the user-supplied `.mobileconfig` plist server-side: whitespace stripping, version normalisation (`1.0` → `1`), server-stamped defaults, top-level UUID/Identifier rewrites, per-payload display-name defaults keyed on `PayloadType`. Trying to *predict* every mutation produces a brittle map of `PayloadType` → server-default lookups that drifts the moment Jamf changes a default. The provider uses **mask-and-compare** instead.

**Strategy.** The plan modifier parses both sides (user input + server-canonical state) and runs the same `maskServerControlledKeys(p)` function across both before comparing. The mask is symmetric and content-blind — the provider never needs to know that `com.apple.notificationsettings` defaults to `"Notifications Payload"`; both sides drop the key, both sides agree.

Mask drops (or empty-normalises) from **both** sides:

- **Top-level always-clobbered**: `PayloadDisplayName` (set from `general.name`), `PayloadIdentifier`, `PayloadUUID` (Jamf assigns lowercase UUIDs).
- **Top-level server-added defaults**: `PayloadEnabled`, `PayloadDescription`, `PayloadRemovalDisallowed`.
- **`PayloadContent[i]` server-augmented**: `PayloadDisplayName`, `PayloadUUID`, `PayloadIdentifier`, `PayloadOrganization`, `PayloadEnabled`, `PayloadDescription`, `AllowUserOverrides`, `VendorConfig`.
- **String values**: recursive leading/trailing whitespace trim (Jamf strips whitespace inside e.g. `Rules[].Comment`).
- **Empty-string normalisation**: `""` on either side compares equal to "absent".

If `inp_masked == srv_masked` the modifier suppresses the diff by setting `plan.payloads = state.payloads`. Otherwise the plan keeps the raw user input and Terraform plans the change.

**Trade-off accepted**: if the user authors a meaningful change to one of the masked keys (e.g. a hand-tuned `PayloadOrganization`), the provider will not detect drift — the server overwrites that value on the next write anyway, so a permanent diff would be the alternative. Document this in the `payloads` attribute `MarkdownDescription`.

**Update-path identifier injection.** On Update, the input builder parses `plan.payloads`, overwrites the top-level `PayloadUUID` and `PayloadIdentifier` with the values from `state.payloads`, and reserialises before PUT. Without this step every Update mints fresh UUIDs server-side and devices treat the update as a fresh profile installation ("ghost profile"). Top-level only is sufficient — nested `PayloadContent[i].PayloadUUID` survives PUT cycles without intervention once stored. Same mechanism as jamf-cli `profileconvert.InjectIdentifiers`.

**Asymmetric envelope `<level>` normalisation.** Classic accepts `<level>Computer</level>` on write but echoes `<level>System</level>` on read; `<level>Computer Level</level>` is silently rejected and defaults to `User`. Use the input-boundary translation pattern from §"Asymmetric server normalisation on type-style discriminator fields" — translate user-facing `Computer Level` / `User Level` to wire `Computer` / `User` on write, map wire `System` / `User` back on read.

Reference implementation: `internal/resources/pro/configuration_profiles/macos_configuration_profile/` (`helpers.go` mask logic, `plan_modifiers.go` plan-time integration, `crud.go` Update path injection, `resource.go` MarkdownDescription disclosing the masked key set to users).

## Error Handling

Use the shared helpers from `internal/common/helpers` rather than rolling your own:

- `helpers.IsNotFoundError(err)` — 404 detection in `Read`/`Delete` operations.
- `helpers.IsServerError(err)` — 5xx detection for retry decisions.
- `helpers.ResolveTimeout(ctx, value, defaultDuration)` — resolve `framework-timeouts` values to a concrete deadline.
- `helpers.ReconcileOptionalBool` / `ReconcileOptionalInt` / `ReconcileOptionalString` / `Reconcile*Pointer` — preserve Terraform null/optional semantics when the API returns zero values.
- `helpers.PreserveStringWhenWireEmpty(wire, current)` — for user-authored string fields where the Classic API echoes empty strings after successful writes. Defends against the masked-error-path bug when the field is a sibling of a `Sensitive` attribute. See §"State builders" rule 4a.
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

The SDK does not currently retain side-by-side versioned functions on regeneration (the upstream generator change requested in [jamfplatform-go-sdk#19](https://github.com/Jamf-Concepts/jamfplatform-go-sdk/issues/19) closed without merging). Practical consequences:

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

Helpers in `internal/common/helpers/ids.go`:

- `IntIDToString(id int64) types.String`
- `StringToIntID(s types.String) (int64, error)`
- `StringValueFromIntPtr(*int) types.String` — for ProClassic SDK pointer IDs.
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

### Shared abstractions — when to extract

Two extraction triggers apply, one for schemas and one for code helpers. They differ because the failure modes differ.

**Schemas — 3-consumer rule.** Many Jamf Pro resources expose similar shapes (scope, site, category, criteria, self-service payload). **Do not extract these into shared schemas upfront** — superficially similar Jamf APIs often differ in field names, ID types (int vs UUID string), and null semantics. Premature abstraction produces helpers with per-resource branching that is harder to read than the original duplication. Refactor trigger: **3 or more** shipped resources with verified-identical shape (same fields, types, null semantics — checked against the SDK structs, not eyeballed). Extract under `internal/common/schemas/`. Reference precedent: `internal/common/scope/` (scope sub-blocks extracted at sub-block granularity; the top-level scope shape stayed per-resource because it diverged across the eight scope-bearing resources).

**Code helpers — 2-consumer rule when logic is non-trivial.** Parsers, normalisation/diff-suppression functions, identifier injectors, and similar non-trivial code with no per-resource branching extract at the **second** consumer, not the third. Two copies of a 200-LOC plist mask or a `WriteOnly` rotation comparator means two places to chase the next server-side mutation; a duplicated bug now lives twice. The schema 3-consumer rule was written for cases where premature abstraction produces ugly branching — that risk does not apply to code helpers that compose without branching.

Canonical example: when `jamfplatform_pro_mobile_device_configuration_profile` lands, lift the mobileconfig `maskServerControlledKeys` + `InjectIdentifiers` helpers from `internal/resources/pro/configuration_profiles/macos_configuration_profile/helpers.go` into `internal/common/profileconvert/` **before** duplicating them into the mobile package. Don't wait for the third.

Trigger summary:

| Kind | Trigger | Destination |
|---|---|---|
| Schema (shape, field set) | 3+ verified-identical SDK shapes | `internal/common/schemas/` (or domain-specific package like `internal/common/scope/`) |
| Code helper (non-trivial, no per-resource branching) | 2 consumers | `internal/common/<topic>/` |
| Trivial code helper (1-line wrapper) | Stays in-resource; extract only on demonstrated need | `internal/common/helpers/` |

### Scope helper

The `<scope>` block of every Jamf Classic-API resource (policies, ebooks, mac applications, mobile device applications, OS X configuration profiles, mobile device configuration profiles, patch policies, restricted software) shares its sub-block target categories — buildings, departments, computers, computer_groups, mobile_devices, mobile_device_groups, network_segments, jss_users, jss_user_groups, ibeacons, and the directory-service name-only siblings. The 3-consumer rule fires at **sub-block granularity**, not at the top-level scope shape (which diverges across the eight scope-bearing resources). Shared helpers live under `internal/common/scope/`.

**Item shape — IDs-only `Set<String>`.** Sub-blocks collapse to a flat `schema.SetAttribute{ElementType: types.StringType, Optional: true}` carrying only the numeric Jamf Pro classic ID (or name string, for the directory-service categories). Server-augmented `<name>` and `<udid>` wire fields are discarded on read; only IDs round-trip through Terraform state. Authoring uses interpolation: `computer_ids = [for c in data.jamfplatform_pro_computers.example: c.id]`. Rationale: confirmed against `Jamf-Concepts/jamf-cli/internal/scope` and `deploymenttheory/terraform-provider-jamfpro/internal/common/shared_schemas/*_scope.go`. The richer alternative — nested `{ id, name }` objects — replays server-derived names back into TF state and forces drift suppression on every refresh; IDs-only sidesteps it.

**Naming convention.**

| Sub-block kind | Suffix | Examples |
|---|---|---|
| ID-bearing | `_ids` | `computer_ids`, `computer_group_ids`, `building_ids`, `department_ids`, `jss_user_ids`, `jss_user_group_ids`, `network_segment_ids`, `ibeacon_ids`, `class_ids` |
| Directory-service name-only | `_names` | `directory_service_or_local_user_names`, `directory_service_user_group_names`, `limit_to_user_group_names` |

Limitations and exclusions share the same attribute names — the wire-shape divergence (limitations user is name-only on wire; exclusions user is id+name on wire) is collapsed at the TF layer because both sides write `<user><name>…</name></user>` and discarding the response-side `<id>` is consistent with Option B.

**Composition pattern.** Per-resource glue assembles the resource's `scope` schema by composing `scope.IDSetAttribute` / `scope.NameSetAttribute` calls. There is **no** top-level `ScopeAttribute()` mega-factory: the eight scope-bearing classic resources (`Policy`, `Ebook`, `MacApplication`, `MobileDeviceApplication`, `OsXConfigurationProfile`, `MobileDeviceConfigurationProfile`, `PatchPolicy`, `RestrictedSoftware`) expose materially different top-level field sets, so a unified factory would either leak unsupported fields or devolve into per-resource branching. The 3-consumer rule fires at sub-block granularity only.

**Cross-field validator — `scope.AllFlagConflictsWith`.** A value-discriminated `validator.Bool` for `all_computers` / `all_mobile_devices` / `all_jss_users` semantics: fires only when the bool is true, attaches one attribute error per populated conflicting Set. Off-the-shelf `boolvalidator.ConflictsWith` triggers on any value and cannot express the "only when true" rule. Resource-specific constraints (e.g. `RestrictedSoftware` rejects `limitations` entirely) stay in the resource package — they are not shared scope logic.

**Omission semantics (load-bearing invariant).** The classic API tolerates absent sections — a `POST` / `PUT` body need not include `<scope>`, `<reboot>`, `<self_service>`, etc. when the caller does not intend to manage them. Go's `encoding/xml` produces this behaviour for free only when the corresponding SDK field is a nil pointer with `,omitempty` on the XML tag. Wire behaviour by builder output:

| Builder assigns | Wire emits |
|---|---|
| `PolicyPost.Scope = nil` | no `<scope>` element |
| `PolicyPost.Scope = &PolicyScope{}` (zero-value) | `<scope></scope>` empty element |
| `PolicyPost.Scope = &PolicyScope{Buildings: &PolicyScopeBuildings{}}` | `<scope><buildings></buildings></scope>` empty parent — **avoid** |
| `PolicyPost.Scope = &PolicyScope{Buildings: &PolicyScopeBuildings{Building: &[]IDName{…}}}` | full populated tree |

Provider input-builder rules:

1. **TF block absent (null) → SDK field nil → wire omits the element.** Default case.
2. **TF block present but empty → leave it nil.** An empty `scope {}` in HCL is semantically "I don't want to manage scope here". `BuildIDSlice` / `BuildNameSlice` return `(nil, nil)` for null, unknown, and empty input.
3. **Sub-block parent pointer is only assigned when at least one child collection is non-empty.** Never emit `&PolicyScopeBuildings{Building: &[]IDName{}}`. Thread `nil` up; if every child is nil, skip the parent assignment.
4. **`all_*` booleans are their own block.** `PolicyScope.AllComputers` is `*bool xml:"all_computers,omitempty"`. `false` marshals as `<all_computers>false</all_computers>` — distinct from omission. Use `helpers.OptionalBoolPointer` so attributes the user did not set collapse to nil.
