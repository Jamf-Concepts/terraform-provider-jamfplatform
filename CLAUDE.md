# Repository Guidelines

## Overview

Terraform provider for Jamf Platform APIs, built on the [Terraform Plugin Framework](https://github.com/hashicorp/terraform-plugin-framework) v1.19.0 with Protocol v6. Go module path: `github.com/Jamf-Concepts/terraform-provider-jamfplatform`.

Four construct types: **resources** (CRUD), **data sources** (read-only lookups), **list resources** (RSQL-filtered streaming), **actions** (fire-and-forget device commands).

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
├── common/
│   ├── availabletitles/ # Shared patch available-titles lookup (patch_external_source, patch_internal_source)
│   ├── criteria/      # Shared smart-group / advanced-search criteria operator vocabulary (device_group, user_group, future searches)
│   ├── files/         # Shared upload-source plumbing for resources that upload file content
│   ├── filters/       # RSQL + classic filter schema/expression builder
│   ├── helpers/       # Type conversions, polling, timeout, state reconciliation, dynamic JSON, IDs, Pro version
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
examples/{provider,resources,data-sources,list-resources,actions}/
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
