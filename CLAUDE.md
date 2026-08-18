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

## Impact alerts — one-paragraph orientation

`internal/common/impact/` produces Jamf Pro's **impact alert notifications** during `terraform plan`: an advisory warning on each object whose scope or payload is changing, reporting how many computers or mobile devices the change reaches. Off by default behind the provider's `impact_alerts` attribute; a nil `*impact.Cache` means disabled, so resources need no flag check. Three channels. Two mirror Jamf's own split — **deployable** (policies, profiles, apps, blueprints, benchmarks) and **scopeable** (groups, classes) — because a resource's `ModifyPlan` cannot see a sibling's, so a plan editing both a group and something scoped to it needs an alert from each side. The third, **policy dependencies** (script, package, printer, Dock item, directory binding, disk encryption configuration), has no counterpart in Jamf Pro: these objects have no scope of their own, so their blast radius is the combined audience of the policies referencing them. Resources wire in via `impact.ReportPlan` (deployable), `impact.ReportMembership` (scopeable) or `impact.ReportDependencyPlan` (dependencies), plus a per-family adapter — a reducer to `impact.Scope` for the first two, and for a dependency only a reader for the object's id and name. The shared Jamf Pro scope block goes through `scope.BuildImpactScope`, which is the single place the narrows/broadens/counts classification lives. Dependencies need a whole-tenant policy sweep, since Jamf Pro has no reverse lookup ("which policies use this script"): lazy, at most once per configured provider instance, self-capped at 5 concurrent reads, and built by `impact.NewCacheWithPolicies` (`NewTenantCache` wires both sources). Alerts are advisory — a tenant that cannot be read yields one notice and never fails a plan. Two rules that are easy to get wrong: a numeric Jamf Pro group id is unique only **within an estate** (see [STYLE_GUIDE.md §Scope helper](STYLE_GUIDE.md#scope-helper)), and group membership is expressed in **device management identifiers**, so a Jamf Pro numeric device/building/department id must be resolved through the inventory before it can be compared with a group's members. User guidance, including the `terraform plan -json` caveat, is in `docs/guides/impact-alerts.md`.

## Jamf Pro resources — one-paragraph orientation

Terraform construct name format: `jamfplatform_pro_<resource>` regardless of whether the SDK source is `pro/` or `proclassic/`. One `jamfplatform.Client` built from `JAMFPLATFORM_*` credentials serves both Platform Services and Pro. Every Pro resource declares an unexported `const minJamfProVersion` and funnels Configure through `providerdata.ConfigurePro` (no hand-rolled boilerplate). Each Pro resource's `crud.go` opens with an SDK-endpoints annotation block (`Status: current. Last reviewed YYYY-MM-DD.`) — Pro / ProClassic only; Platform Services resources are exempt. Full rules: [STYLE_GUIDE.md §Jamf Pro Resource Naming](STYLE_GUIDE.md#jamf-pro-resource-naming), §Minimum Jamf Pro version check, §Endpoint adoption & migration policy. Workflow for adding a Pro resource (incl. SDK-comparison + ProClassic payload audit gate): [CONTRIBUTING.md §Adding a Jamf Pro Resource](CONTRIBUTING.md#adding-a-jamf-pro-resource).

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
| `test` | Unit tests (excludes `acceptance` build tag) |
| `testacc` | Acceptance tests (sets `TF_ACC=1`, requires tenant) |
| `testacc-run` | Targeted acc rerun (`RUN=<regex> PKG=<path>`) |

Before committing: `make fix fmt lint test`. Then `make generate` if any schema description or example changed.

## Environment Variables

- `JAMFPLATFORM_BASE_URL` — `https://us.apigw.jamf.com` / `eu.apigw.jamf.com` / `apac.apigw.jamf.com`.
- `JAMFPLATFORM_CLIENT_ID` / `JAMFPLATFORM_CLIENT_SECRET` — API client credentials.
- Acceptance tests additionally require `TF_ACC=1` (set automatically by `make testacc`).

## Copyright headers

Every Go file carries `// Copyright Jamf Software LLC <year>` + `// SPDX-License-Identifier: MPL-2.0`. Managed by `copywrite` via `make generate`. 2026
