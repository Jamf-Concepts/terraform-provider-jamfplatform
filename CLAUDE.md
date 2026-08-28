# Repository Guidelines

## Overview

Terraform provider for Jamf Platform APIs, built on the [Terraform Plugin Framework](https://github.com/hashicorp/terraform-plugin-framework) v1.19.0 with Protocol v6. Go module path: `github.com/Jamf-Concepts/terraform-provider-jamfplatform`.

Five construct types: **resources** (CRUD), **data sources** (read-only lookups), **list resources** (RSQL-filtered streaming), **actions** (fire-and-forget device commands), and **functions** (offline provider-defined functions under the `jamfplatform::` namespace — no API client, no provider config).

The Jamf Platform API client is the external Go SDK `github.com/Jamf-Concepts/jamfplatform-go-sdk` (package `jamfplatform`). Not vendored.

## Companion Docs (authoritative on conflict)

- `STYLE_GUIDE.md` — Go style, file conventions, schema rules, Jamf Pro naming/version/endpoint policies, scope helper, profile payload normalisation, ID/import conventions.
- `TESTING.md` — test categories, commands, build tags, CI.
- `CONTRIBUTING.md` — contribution workflow, Pro-resource workflow (incl. ProClassic SDK payload audit), release-versioning policy.
- `README.md` — user-facing provider usage.
- `spike/JAMF_PRO_INVENTORY.md` — gitignored Pro SDK namespace adoption tracker.
- `spike/PRO_ROLLOUT_PLAN.md` — gitignored Pro rollout planning doc.

## Project Structure

```
internal/
├── provider/          # Provider config, registration, logging
├── providerdata/      # Shared Data{} carrying SDK client + cached Pro version
├── resources/
│   ├── blueprints/    # blueprint/, blueprints/, component/, components/    (Platform Services)
│   ├── cbengine/      # benchmark/, benchmarks/, baselines/, rules/         (Platform Services)
│   ├── device/        # Single device data source                            (Platform Services)
│   ├── device_group/  # Resource, data source, list resource                 (Platform Services)
│   ├── device_groups/ # Plural data source                                   (Platform Services)
│   ├── devices/       # Plural data source                                   (Platform Services)
│   ├── security_cloud/ # Jamf Security Cloud — flat single tier                (Security Cloud)
│   │                  #   dns_zone/, ztna_gateway/, ztna_grouped_gateway/, ztna_shared_gateways/
│   │                  #   (folder name = Terraform slug minus `jamfplatform_security_cloud_`)
│   └── pro/           # Jamf Pro resources — flat single tier: every leaf package sits directly under pro/
│                      #   (folder name = Terraform slug minus `jamfplatform_pro_`, snake_case). No domain tier.
│                      #   ~115 packages, fully flat (e.g. the five PKI constructs are pki_adcs/, pki_venafi/, pki_digicert/, … — no pki/ grouping dir).
├── actions/
│   ├── device/        # erase, restart, shutdown, unmanage                       (Platform Device Actions API)
│   └── pro/           # managed_software_updates (plan + abandon), maintenance/ (flush_policy_logs, redeploy_management_framework), mdm/ (13 MDM commands), patch/ (retry_patch_policy_logs)   (Jamf Pro)
├── functions/         # Provider-defined functions (offline; no SDK client, no provider config)
│   ├── mobileconfig/       # mobileconfig(profile) — build a full .mobileconfig from HCL payloads; also holds the shared Assemble core
│   └── mcx_forced_payload/ # mcx_forced_payload(domain, prefs) — MCX "Custom Settings" envelope; thin wrapper over mobileconfig.Assemble
├── common/
│   ├── appleprofiles/ # Apple configuration profile schemas (generated table + plan-time payload validation)
│   ├── availabletitles/ # Shared patch available-titles lookup (patch_external_source, patch_internal_source)
│   ├── criteria/      # Shared smart-group / advanced-search criteria operator vocabulary (device_group, user_group, future searches)
│   ├── files/         # Shared upload-source plumbing for resources that upload file content
│   ├── filters/       # RSQL + classic filter schema/expression builder
│   ├── helpers/       # Type conversions, polling, timeout, state reconciliation, dynamic JSON, IDs, Pro version
│   ├── impact/        # Plan-time impact alerts — device counts for scope changes (see §Impact alerts)
│   ├── invitationcommon/ # Shared enrollment-invitation helpers (computer_invitation, mobile_device_invitation)
│   ├── ldapgroups/    # Directory-service (LDAP / cloud-IdP) group resolution + scope preflight validation
│   ├── payloadhelpers/ # mobileconfig mask / compare / identifier injection (macos/mobile_device_configuration_profile)
│   ├── planmodifiers/ # Shared Terraform Plugin Framework plan modifiers
│   ├── plisthelpers/  # Generic plist (Apple property list) parsing / normalisation helpers
│   ├── scope/         # Classic scope sub-block factories + builders + validators (see STYLE_GUIDE §Scope helper)
│   └── validators/    # Shared Terraform Plugin Framework validators (unique-string-field across collections)
└── testhelpers/       # Acceptance fixtures (provider factories, real client, mock server)
tools/                 # go:generate entrypoint (copywrite, terraform fmt, tfplugindocs)
local-testing/         # Manual API request workflows for development (gitignored)
examples/{provider,resources,data-sources,list-resources,actions,functions}/
docs/                  # Auto-generated provider documentation — do not hand-edit
```

Each leaf resource folder mirrors the file split in [STYLE_GUIDE.md §Resource Package File Conventions](STYLE_GUIDE.md#resource-package-file-conventions).

### Reference implementations (copy from these)

| Pattern | Reference |
|---|---|
| Complex CRUD with state upgrader + nested payload sub-package | `internal/resources/blueprints/blueprint/` |
| Simple Pro CRUD (+ list + data sources) | `internal/resources/pro/category/` |
| Pro CRUD with lossy-PUT canonicalisation + snake_case mapping | `internal/resources/pro/script/` |
| Pro singleton settings | `internal/resources/pro/self_service_plus_settings/` |
| ProClassic CRUD | `internal/resources/pro/site/` (and `network_segment/`) |
| Scope-bearing classic resource | `internal/resources/pro/policy/` |
| Classic configuration profile (mobileconfig payload diff suppression) | `internal/resources/pro/macos_configuration_profile/` |
| Plaintext secret with `WriteOnly + _wo_version` | `internal/resources/pro/directory_binding/` |
| Classic-CRUD resource with a v2 side-channel (extension-attribute accept) | `internal/resources/pro/patch_software_title/` |
| Positional id-less nested lists + opt-out sub-collections (omit=retain/`[]`=clear) + Computed nested collections as `types.List` | `internal/resources/pro/licensed_software/` |
| Create-only immutable upload (server rejects every PUT once the blob exists → all attrs RequiresReplace + no-PUT Update) | `internal/resources/pro/mobile_device_provisioning_profile/` |
| Classic XML merge PUT where empty clears (always-emit scalars / clear-by-omission) + Location/Purchasing blocks + read-only attachments (bearer-auth-refused upload) | `internal/resources/pro/mobile_device_enrollment_profile/` |
| Provider-defined function (offline; `types.Dynamic` decode + shared core) | `internal/functions/mobileconfig/` |
| Jamf Security Cloud CRUD (+ DS singular/plural + list; no tenant version gate) | `internal/resources/security_cloud/dns_zone/` |
| Two-form resource where the form is immutable and derived from a block's presence | `internal/resources/security_cloud/ztna_gateway/` |
| Read-only server catalogue, plural DS only (no per-id endpoint) | `internal/resources/security_cloud/ztna_shared_gateways/` |

## Impact alerts — one-paragraph orientation

`internal/common/impact/` produces Jamf Pro's **impact alert notifications** during `terraform plan`: an advisory warning on each object whose scope or payload is changing, reporting how many computers or mobile devices the change reaches. Off by default behind the provider's `impact_alerts` attribute; a nil `*impact.Cache` means disabled, so resources need no flag check. Three channels. Two mirror Jamf's own split — **deployable** (policies, profiles, apps, blueprints, benchmarks) and **scopeable** (groups, classes) — because a resource's `ModifyPlan` cannot see a sibling's, so a plan editing both a group and something scoped to it needs an alert from each side. The third, **policy dependencies** (script, package, printer, Dock item, directory binding, disk encryption configuration), has no counterpart in Jamf Pro: these objects have no scope of their own, so their blast radius is the combined audience of the policies referencing them. Resources wire in via `impact.ReportPlan` (deployable), `impact.ReportMembership` (scopeable) or `impact.ReportDependencyPlan` (dependencies), plus a per-family adapter — a reducer to `impact.Scope` for the first two, and for a dependency only a reader for the object's id and name. The shared Jamf Pro scope block goes through `scope.BuildImpactScope`, which is the single place the narrows/broadens/counts classification lives. Dependencies need a whole-tenant policy sweep, since Jamf Pro has no reverse lookup ("which policies use this script"): lazy, at most once per configured provider instance, self-capped at 5 concurrent reads, and built by `impact.NewCacheWithPolicies` (`NewTenantCache` wires both sources). Alerts are advisory — a tenant that cannot be read yields one notice and never fails a plan. Two rules that are easy to get wrong: a numeric Jamf Pro group id is unique only **within an estate** (see [STYLE_GUIDE.md §Scope helper](STYLE_GUIDE.md#scope-helper)), and group membership is expressed in **device management identifiers**, so a Jamf Pro numeric device/building/department id must be resolved through the inventory before it can be compared with a group's members. User guidance, including the `terraform plan -json` caveat, is in `docs/guides/impact-alerts.md`.

## Apple profile schemas — one-paragraph orientation

`internal/common/appleprofiles/` carries a generated table of Apple's configuration profile payload
keys, built from the `mdm/profiles` directory of apple/device-management by `make apple-profiles`
(never part of `make generate` — it needs a network clone, and upstream only moves a few times a
year). Jamf's blueprints service validates a stored legacy payload against the same vocabulary, and
wire probing established exactly how: an unknown **payload type** is rejected and matched
case-sensitively; an unknown **key** is silently discarded; a key differing only in case is silently
stored under Apple's spelling; a wrong value type or a missing required key fails the write; enum and
range constraints are **not** enforced (`AlertType: 99` stores fine), so the table does not check
them either. `appleprofiles.Validate` turns those rules into `Problem` values, and
`Problem.Advisory()` says how far to trust each one — a name the table does not recognise is a
warning, because a key Apple added after the snapshot looks identical to one that never existed;
everything else is an error, because Jamf refuses the write. Descent stops at a free-form
(wildcard) dictionary: everything under an MCX preference domain
(`com.apple.ManagedClient.preferences`, the "Custom Settings" envelope) is passthrough, and Jamf
stops validating there too. The blueprint resource wires this in through `validators.go` for both
`component_blocks[].legacy_payloads` and the deprecated top-level `legacy_payloads`. Freshness is a
scheduled concern, not a plan-time one: `.github/workflows/apple-profiles.yml` regenerates monthly
and opens a pull request, the same reviewable-PR pattern Dependabot uses here.

## Jamf Security Cloud resources — one-paragraph orientation

Terraform construct name format: `jamfplatform_security_cloud_<resource>`; Go package
`internal/resources/security_cloud/<resource>/`, flat single tier like `pro/`. All five
Security Cloud API namespaces (`jsc-categories`, `jsc-dns`, `jsc-ztna`,
`securitycloud-devices`, `uem-connect`) are generated into one SDK package,
`jamfplatform/securitycloud`, and every method routes through the unified
`/api/securitycloud` prefix — wire-verified in production EU on 2026-08-27 for the DNS
surface under a tenant-scoped integration. Configure goes through
`providerdata.ConfigureSecurityCloud`, **not** `ConfigurePro`: Security Cloud is
continuously deployed with no customer-tenant version, and a tenant can hold it without
holding Jamf Pro, so a Pro version fetch would be both meaningless and fatal. Two things
differ from every other namespace and are easy to get wrong. First, **entitlement is not
authentication** — a valid integration can still be refused with `403 NOT_ENTITLED`, so
resources translate that code into a named diagnostic instead of surfacing the raw error.
Second, **cross-namespace references are server-enforced in both directions**: a DNS
zone's name servers each name a gateway by ID — a shared, dedicated or grouped gateway, all
three accepted — and a zone cannot be written before its gateway exists
(`422 GATEWAY_NOT_FOUND`), so that diagnostic points at `name_servers` rather than the zone;
conversely a gateway that anything still references refuses to be deleted with a bare
`409 CONFLICT` naming nothing, which is a Terraform destroy-ordering trap and gets its own
diagnostic. Two shapes recur across the namespace and are worth knowing before reading any of
it: a **cipher/algorithm field is an array the server accepts exactly one element in**, so it
is modelled as a single string and collapsed at the boundary; and an **enum violation returns
`400 [INVALID_FIELD] Request body is missing or malformed.`** with no field and no value,
which is why every enum is validated at plan time from the SDK's own generated `*Values()`
helper rather than a restated list. Acceptance tests gate on
`testhelpers.AccPreCheckSecurityCloud`, which requires the operator to *declare* that the
configured scope is a Security Cloud one (`JAMFPLATFORM_SECURITY_CLOUD_{ENVIRONMENT,TENANT}_ID`,
matching the scope in use) and skips otherwise — a Pro-only acceptance tenant is a
legitimate environment, not a failure. Full rules:
[STYLE_GUIDE.md §Jamf Security Cloud Resource Naming](STYLE_GUIDE.md#jamf-security-cloud-resource-naming).

## Jamf Pro resources — one-paragraph orientation

Terraform construct name format: `jamfplatform_pro_<resource>` regardless of whether the SDK source is `pro/` or `proclassic/`. One `jamfplatform.Client` built from `JAMFPLATFORM_*` credentials serves both Platform Services and Pro. Every Pro resource declares an unexported `const minJamfProVersion` and funnels Configure through `providerdata.ConfigurePro` (no hand-rolled boilerplate). Each Pro resource's `crud.go` opens with an SDK-endpoints annotation block (`Status: current. Last reviewed YYYY-MM-DD.`) — Pro / ProClassic only; Platform Services resources are exempt. Full rules: [STYLE_GUIDE.md §Jamf Pro Resource Naming](STYLE_GUIDE.md#jamf-pro-resource-naming), §Minimum Jamf Pro version check, §Endpoint adoption & migration policy. Workflow for adding a Pro resource (incl. SDK-comparison + ProClassic payload audit gate): [CONTRIBUTING.md §Adding a Jamf Pro Resource](CONTRIBUTING.md#adding-a-jamf-pro-resource).

## API integration scope — one-paragraph orientation

Jamf offers three scopes when an API integration is created, and the provider mirrors all three: **Platform environment** (a group of tenants across product types — the *preferred* scope, `environment_id` → `X-Environment-Id`), **Tenant** (a single Jamf Pro / School / Protect / Security Cloud tenant — Jamf's own words are "legacy method for targeting integrations without a platform environment", `tenant_id` → `X-Tenant-Id`), and **Organization management** (SSO, AI Governance and similar organization-level resources — no scope header at all; the gateway resolves the context from the access token). Since SDK v0.17.0 the scope travels in a header rather than the URL path, set by `WithEnvironmentID` / `WithTenantID`. So `environment_id` and `tenant_id` are **mutually exclusive and both optional** — an integration targets one, and supplying the other is refused with `403 OWNERSHIP_FORBIDDEN` even when both IDs belong to the same customer. `internal/provider/scope.go` resolves which is in play (config beats environment; both-at-once is an error either way; a shadowed env var warns) and selects the SDK option accordingly; `providerdata.New` then reads the scope back off the built client via `Client.Scope()` (SDK v0.18.0), so the gate can never disagree with the header the client actually sends. Whether a given construct can be reached under that scope is then enforced **per construct**, not once in provider Configure, via `providerdata.RequireScope` in `internal/providerdata/scope.go` — because the answer differs per API family and is about to differ more: Jamf Pro works under either scope (gated once inside `configureSub`, covering every `pro/` package and Pro action), Security Cloud likewise works under either and is gated once inside `ConfigureSecurityCloud`, while Blueprints and Compliance Benchmarks become environment-only at the Platform API GA, at which point their call sites drop `ScopeTenant`. Pass the allowed kinds in preference order, environment first — it is the order the diagnostic lists them in. Organization scope is currently rejected everywhere, deliberately: it turns an opaque gateway failure mid-apply into a named diagnostic at Configure, and the organization-level constructs that would accept it are not built yet.

## Tooling

- Go >= 1.26, Terraform >= 1.13.0.
- `GNUmakefile` is the canonical entrypoint. Default target: `fmt lint install generate`.
- Releases built with goreleaser (`goreleaser.yml`).
- `make generate` → `tools/tools.go`: copyright headers (`hashicorp/copywrite`), `terraform fmt -recursive ../examples/`, provider docs (`hashicorp/terraform-plugin-docs`).

| Target | Description |
|---|---|
| `build` | Build the provider |
| `install` | Build and install locally |
| `fmt` | `gofmt -s -w -e .` |
| `fix` | `go fix ./...` — rewrites deprecated API usages |
| `lint` | `golangci-lint run` |
| `generate` | Copyright headers + `terraform fmt examples/` + docs |
| `apple-profiles` | Regenerate `internal/common/appleprofiles/profiles.json` from apple/device-management (network; not part of `generate`) |
| `test` | Unit tests (excludes `acceptance` build tag) |
| `test-scripts` | Unit tests for `scripts/acctargets` (behind the `acctargets` build tag, so `go test ./...` misses it) |
| `testacc` | Acceptance tests (sets `TF_ACC=1`, requires tenant) |
| `testacc-run` | Targeted acc rerun (`RUN=<regex> PKG=<path>`) |

Before committing: `make fix fmt lint test`. Then `make generate` if any schema description or example changed.

## Environment Variables

- `JAMFPLATFORM_BASE_URL` — `https://us.api.jamfcloud.com` / `eu.api.jamfcloud.com` / `apac.api.jamfcloud.com`. The gateway root, host only: it serves `/auth/token` and every namespace at the root, so a `/api` path breaks authentication. The beta `{region}.apigw.jamf.com` was retired at the Platform API GA.
- `JAMFPLATFORM_CLIENT_ID` / `JAMFPLATFORM_CLIENT_SECRET` — API client credentials.
- `JAMFPLATFORM_ENVIRONMENT_ID` — platform-environment scope, sent as `X-Environment-Id`. **Preferred.**
- `JAMFPLATFORM_TENANT_ID` — tenant scope, sent as `X-Tenant-Id`. **Legacy.** Mutually exclusive with the above; both are optional — see §API integration scope.
- `JAMFPLATFORM_SECURITY_CLOUD_ENVIRONMENT_ID` / `JAMFPLATFORM_SECURITY_CLOUD_TENANT_ID` — **acceptance tests only**. Declares that the configured scope belongs to a Jamf Security Cloud tenant; must equal the corresponding `JAMFPLATFORM_*` value. Unset or mismatched and every Security Cloud acceptance test skips — which is CI's current state, deliberately. The ZTNA gateway tests additionally require the *tenant* form: `tenantIds` is mandatory on a gateway and no API exposes an environment's tenants, so an environment-scoped run cannot supply one. See [TESTING.md](TESTING.md).
- Acceptance tests additionally require `TF_ACC=1` (set automatically by `make testacc`), and one of the two scope variables.

## Copyright headers

Every Go file carries `// Copyright Jamf Software LLC <year>` + `// SPDX-License-Identifier: MPL-2.0`. Managed by `copywrite` via `make generate`. 2026
