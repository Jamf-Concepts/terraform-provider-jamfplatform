---
page_title: "Preparing for the Platform API GA"
description: |-
  What changes when the Jamf Platform API leaves beta: new credentials, a new base URL, environment scope, and the constructs that were removed.
---

# Preparing for the Platform API GA

The Jamf Platform API is leaving public beta and will reach general availability shortly. The
date will be announced separately.

> **This guide is provisional and may change without notice.** The Platform API is still moving
> ahead of GA, and a late change to it changes this page: the constructs listed as removed, the
> attribute and scope behaviour, and the version numbers quoted throughout are all subject to
> revision. Re-read it against the release you are actually upgrading to rather than working from
> a copy taken earlier.

**Action is needed in every configuration built against the public beta.** Nothing carries over
untouched: the gateway host, the credentials and the scope attribute all change, and several
constructs have been removed. Upgrade promptly once GA is announced, as the beta gateway is
retired at that point and all provider versions prior to `v0.29.0-rc.4` are bound to it.

Each section below states the action required, or says explicitly that none is.

## Acknowledgements

Thank you to everyone who used this provider during the Platform API public beta. Bug reports and
feedback from beta participants shaped a substantial part of what ships at GA, including the
environment scope, proxy support and a range of resource behaviour. Continued feedback is welcome.

## Installing this pre-release

Terraform does not select a pre-release version automatically. With no `version` constraint, or
with a range such as `~> 0.29`, Terraform resolves to the latest stable release, currently
`v0.28.1`; `terraform init -upgrade` will not move a configuration onto a release candidate
either. The version must be named exactly:

```hcl
terraform {
  required_providers {
    jamfplatform = {
      source  = "Jamf-Concepts/jamfplatform"
      version = "0.29.0-rc.6"
    }
  }
}
```

Run `terraform init -upgrade` after editing the constraint.

A stable `v0.29.0` will follow shortly after GA, but not simultaneously with it. Until it is
released, an explicit pre-release constraint is required: `v0.28.1` remains the latest stable
release and is bound to the beta gateway, so an unconstrained configuration, or one constrained to
a range, resolves to a version that cannot reach the API. Relax the constraint to `~> 0.29`, or
remove it, once the stable release is available.

This release can be adopted before GA, with no credential work to do so: an existing public-beta
API integration authenticates against the GA host, and beta credentials remain valid until GA.
Make the `base_url` change and the state removals described under [Action needed](#action-needed),
and only the credential replacement is left outstanding at GA, with the remaining changes already
validated against your own configuration.

## Action needed

In summary, per workspace. The first group applies whenever this release is adopted, including
during the public beta; the second only at GA.

**On upgrading to this release:**

1. **Remove the deleted resources from state.** After the upgrade, a workspace still holding one of
   them cannot produce a plan of any kind. See [Removed constructs](#removed-constructs).
2. **Set `base_url` to the GA gateway in the same change as the provider upgrade.** The host and
   the provider version are paired. See [Base URL](#base-url).
3. **Extend the integration's permissions where a workspace uses
   `jamfplatform_pro_patch_software_title`.** Its data source, list resource and `terraform import`
   now read the tenant's patch source catalogues, so two read permissions must be added. See
   [Attribute removals and deprecations](#attribute-removals-and-deprecations).
4. Take a state backup, as for any breaking provider upgrade.

Existing beta credentials and `tenant_id` continue to work through this upgrade, and nothing else
is required to adopt the release before GA.

**At GA:**

5. **Register a replacement API integration and update the provider credentials.** Record the
   permissions held by each beta integration beforehand, so the equivalents can be selected. See
   [API integration credentials](#api-integration-credentials).
6. **Replace `tenant_id` with `environment_id`.** Register the replacement as
   environment-scoped unless single-product access is deliberately what you want: tenant scope is
   the legacy option, costs one integration per tenant per product, and cannot hold the blueprint
   or compliance-benchmark permissions at all. See [Scope](#scope).

## Removed constructs

Several endpoints were unpublished at GA. The provider cannot call an endpoint that is no longer
available, so the constructs built on those endpoints have been removed.

### Managed resources (state edit required)

| Removed resource | Reason | Replacement |
|---|---|---|
| `jamfplatform_pro_api_client` | `/v1/api-integrations` unpublished — security hardening | Jamf Account, Platform API integrations UI |
| `jamfplatform_pro_api_role` | `/v1/api-roles` unpublished — security hardening | Jamf Account, Platform API integrations UI |
| `jamfplatform_pro_app_installer` | App Installer endpoints unpublished | None in this provider |
| `jamfplatform_pro_app_installer_settings` | App Installer endpoints unpublished | None in this provider |

The API client and API role endpoints were unpublished to close a privilege-escalation path: an
API integration was able to create a client or role holding privileges the creating integration
did not itself hold.

**Action needed: remove the state entries.** All four shipped in `v0.28.1`, so existing state
files contain them. Terraform requires a schema
for every resource type present in state before it can complete `plan`, `apply` or `destroy`. Once
the type is no longer implemented by the provider, every operation in that workspace fails,
including operations unrelated to these resources. Deleting the `resource` block is not sufficient,
because the state entry remains. The resulting error is:

```
Error: no schema available for jamfplatform_pro_api_client.example while reading state;
this is a bug in Terraform and should be reported
```

Despite the wording, this is not a Terraform defect. Remove the state entries with
[`terraform state rm`](https://developer.hashicorp.com/terraform/cli/commands/state/rm), which is
a state-only operation requiring no schema and therefore works both before and after the upgrade:

```sh
# identify the affected addresses
terraform state list | grep -E 'jamfplatform_pro_(api_client|api_role|app_installer)'

# preview the removal without modifying state
terraform state rm -dry-run 'jamfplatform_pro_api_client.example'

# remove each address reported by the list command
terraform state rm 'jamfplatform_pro_api_client.example'
```

A single address removes every instance of that resource, including instances created by `count`
or `for_each`. A local backend writes a `terraform.tfstate.<timestamp>.backup` file before making
the change; take an equivalent backup manually when using a remote backend.

`terraform state rm` only ends Terraform's management of the object. The API client, role, App
Installer deployment or settings object itself remains in place in Jamf Pro.

### Data sources and list resources (configuration edit only)

**Action needed: delete the blocks.** These hold no state, so removing them from the
configuration is the whole of it.

Data sources: `jamfplatform_pro_api_client`, `jamfplatform_pro_api_clients`,
`jamfplatform_pro_api_role`, `jamfplatform_pro_api_roles`,
`jamfplatform_pro_api_role_privileges`, `jamfplatform_pro_app_installer`,
`jamfplatform_pro_app_installers`, `jamfplatform_pro_app_installer_settings`,
`jamfplatform_pro_app_installer_title`, `jamfplatform_pro_app_installer_titles`.

List resources: `jamfplatform_pro_api_client`, `jamfplatform_pro_api_role`,
`jamfplatform_pro_app_installer`.

`jamfplatform_pro_api_role_privileges` has no replacement, as no endpoint serves the privilege
list. The privileges themselves are retained in a new form, described under
[API integration credentials](#api-integration-credentials).

### Actions (configuration edit only)

**Action needed: delete the blocks.** `POST /v2/mdm/commands` was unpublished, and it was the
only means of queuing an MDM command through the Platform API. Actions hold no state, so removing
the `action` block and any `lifecycle { action_trigger }` referring to it is sufficient.

Removed: `jamfplatform_pro_device_lock`, `jamfplatform_pro_enable_lost_mode`,
`jamfplatform_pro_disable_lost_mode`, `jamfplatform_pro_play_lost_mode_sound`,
`jamfplatform_pro_enable_remote_desktop`, `jamfplatform_pro_disable_remote_desktop`,
`jamfplatform_pro_clear_restrictions_password`, `jamfplatform_pro_clear_passcode`,
`jamfplatform_pro_delete_user`, `jamfplatform_pro_log_out_user`,
`jamfplatform_pro_unlock_user_account`, `jamfplatform_pro_set_auto_admin_password`,
`jamfplatform_pro_trigger_enhanced_log_collection`,
`jamfplatform_pro_cancel_enhanced_log_collection`.

Three MDM actions are unaffected and continue to function, as none of them used the unpublished
endpoint: `jamfplatform_pro_send_blank_push`, `jamfplatform_pro_renew_mdm_profile` and
`jamfplatform_pro_flush_mdm_commands`.

The underlying capability is retained. The Jamf Pro Classic API continues to serve these commands
on a different request shape, and rebuilding these actions over Classic is planned, subject to
endpoint availability.

## API integration credentials

**Action needed at GA: register a replacement integration and update the provider credentials.**
Beta API integration credentials remain valid until GA, including against the GA gateway host, and
are revoked at GA. A beta client cannot be migrated to a GA client, and both the permission model
and the endpoint paths have changed.

Credentials are registered in the Jamf Account Platform API integrations UI. A single OAuth 2.0
authentication model now covers the whole platform, in place of one credential set per product.
Step-by-step instructions are published at GA:

**[Getting started with the Platform API](https://developer.jamf.com/platform-api/reference/getting-started-with-platform-api)**

Refer to that page for how to register an integration and where to obtain its client ID and
secret.

**Permissions are organised by capability and action** rather than by product privilege list. The
API names take the form `compliance-benchmarks:create` or `device-groups:read`, and Jamf Account's
permission picker presents the same thing as a named permission with a checkbox per action.
Resource, data source, list resource and action pages in this documentation carry a **Required Jamf
permissions** table written the way the picker reads — the section, the permission name, the boxes
to tick, and the API capability behind them. Use those tables to select permissions for the
replacement integration, granting only what the constructs in use require.

Two consequences of the new model are worth knowing before granting anything. An action covers
only itself, so an integration that reads a record before modifying it needs the read action as
well as the update action. And the pre-GA computer and mobile privilege pairs have collapsed into
single device-level permissions — `devices`, `device-groups`, `extension-attributes`,
`configuration-profiles`, `enrollment-invitations`, `advanced-device-searches` and
`prestage-enrollments` — so a computers-only integration is no longer expressible. Jamf's
**[Jamf Pro permissions map](https://developer.jamf.com/platform-api/reference/jamf-pro-permissions-map)**
is the reference for the full mapping, including the reverse case: the old computer and mobile
command privileges split into `device-actions` and `destructive-device-actions`, the latter
covering erase, unmanage and remove MDM profile.

**Three scope levels are available.** A *platform environment* is a group of tenants across
product types, and is the scope to prefer for two concrete reasons: one environment-scoped
integration covers the whole group, and it is the only scope on which the blueprint and
compliance-benchmark permissions can be selected. A *tenant* scope targets a single Jamf Pro, Jamf
School, Jamf Protect or Jamf Security Cloud tenant. Jamf describes it as the legacy method of
targeting an integration without a platform environment, and it is one integration per tenant per
product — a Jamf Pro tenant and a Jamf Protect tenant are two integrations to create and two
credential pairs to rotate. Treat it as the exception, for a deliberately single-product
integration, rather than the default. An *organization management* scope reaches organization-level
resources — the `jamfplatform_account_*` family — and is the only scope that reaches them; it is
configured by setting *neither* `environment_id` nor `tenant_id`, since the gateway resolves the
organization from the access token. Every resource and data source reports the scope it needs when
it is configured, so pointing an integration at a family it cannot reach is named at plan time
rather than failing mid-apply. The scope selected determines the provider configuration described
under [Scope](#scope).

## Base URL

**Action needed: change the host.** The gateway host and the provider version are paired in both
directions:

| Provider version | `base_url` |
|---|---|
| `v0.29.0-rc.3` and earlier | `https://{region}.apigw.jamf.com` — the beta host. The GA host is not supported on these versions. |
| `v0.29.0-rc.4` and later | `https://{region}.api.jamfcloud.com` — required. The beta host is retired at GA. |

Change the host and upgrade the provider in a single change. Neither the GA host on an earlier
version nor the beta host on `v0.29.0-rc.4` or later is a supported combination. The GA host is
already live and accepts a public-beta API integration, so the change can be made and verified
before GA, independently of the credential replacement.

```hcl
provider "jamfplatform" {
  base_url = "https://us.api.jamfcloud.com" # or eu., or apac.
}
```

The value may also be supplied through `JAMFPLATFORM_BASE_URL`. The region remains mandatory.

Supply the host only. The gateway serves the token endpoint and every API namespace at the root,
and the `/api` path segment used during the beta has been dropped. A request carrying it receives
the gateway's `404 page not found` response rather than a JSON error.

## Scope

**Action needed at GA for most configurations: replace `tenant_id` with `environment_id`.** Not
before then — an existing tenant-scoped beta integration keeps working, including against the GA
gateway host, so this change belongs with the credential replacement rather than with the provider
upgrade.

The scope an API integration targets now travels in a request header rather than a URL path, and
the provider has gained an `environment_id` attribute alongside `tenant_id`. Public-beta
integrations were tenant-scoped, so most configurations in use today set `tenant_id` or export
`JAMFPLATFORM_TENANT_ID`. A GA replacement integration will in most cases be environment-scoped,
so registering the replacement is accompanied by a provider configuration change:

```hcl
provider "jamfplatform" {
  base_url = "https://eu.api.jamfcloud.com"
  # tenant_id      = var.jamf_tenant_id
  environment_id = var.jamf_environment_id
}
```

Both attributes may also be supplied through `JAMFPLATFORM_ENVIRONMENT_ID` and
`JAMFPLATFORM_TENANT_ID`. They are mutually exclusive and both optional: an integration targets
one or the other, `environment_id` is preferred, and `tenant_id` is the legacy method of targeting
integrations without a platform environment. A tenant-scoped GA integration remains valid for
single-product access, and `jamfplatform_pro_*` and `jamfplatform_security_cloud_*` work under
either scope. Three sets of constructs are out of its reach, for three different reasons:

- **AI Governance** is refused by the provider, at configure time, with a diagnostic naming the
  construct. `jamfplatform_ai_governance_policy` and the tool catalogue require environment scope.
- **Blueprints and compliance benchmarks** are refused by Jamf. Their permissions cannot be
  selected when a tenant-scoped integration is created in Jamf Account at GA, so such an
  integration can never hold them and the calls fail with `403 BAD_PERMISSIONS`. The provider
  cannot pre-empt this, as a permission absent from the integration is indistinguishable from any
  other privilege gap — so choose environment scope if the configuration manages either.
- **Jamf Account** requires a third scope, *organization management*, which neither of these
  attributes selects. `jamfplatform_account_*` is the only family that scope reaches, and it is the
  only scope that reaches the family; the provider refuses each direction at configure time with a
  diagnostic naming the construct. Configure it by setting **neither** `environment_id` nor
  `tenant_id` and exporting neither variable — the gateway resolves the organization from the access
  token, so no identifier is supplied. An organization-scoped integration therefore belongs in its
  own provider block, aliased, or its own workspace.

Three failure modes follow from the change, in decreasing order of how easily they are diagnosed:

- Retaining `tenant_id` alongside `environment_id` is rejected at configure time with a
  `Conflicting API Integration Scope` error. The old attribute must be removed, not left in place.
- A `JAMFPLATFORM_TENANT_ID` still exported in CI is ignored, with a warning naming it, when
  `environment_id` is set in the provider block. Where neither attribute is set in the provider
  block and both environment variables are exported, configuration fails with the same conflict
  error. Unset the tenant variable as part of the cutover.
- Supplying the identifier that does not correspond to how the integration was created is refused
  with `403 OWNERSHIP_FORBIDDEN`, even where both identifiers belong to the same customer.

Where a configuration must supply the Jamf Pro tenant identifier to another Jamf product — the
case this was built for is `jamfplatform_security_cloud_uem_connect` — the
`jamfplatform_pro_tenant_id` data source resolves it from the configured scope, removing the need
to transfer the identifier between consoles manually.

## Attribute removals and deprecations

### Removed: `target_device_group` on `jamfplatform_cbengine_benchmark`

**Action needed if the attribute is in use: switch to `target_device_groups`**, a set of device
group identifiers:

```hcl
resource "jamfplatform_cbengine_benchmark" "example" {
  # target_device_group  = jamfplatform_device_group.macs.id
  target_device_groups = [jamfplatform_device_group.macs.id]
}
```

State migrates automatically. The provider folds a singular value into the set during the state
upgrade, so the benchmark continues to target the same device group; the removal requires a
configuration edit only, never a re-scope. The attribute has also been removed from the
corresponding data source.

### Removed: `category_name` and `site_name` on `jamfplatform_pro_patch_software_title`

**Action needed if either attribute is referenced: read the name from the object that owns it.**
Both were read-only display names, and the Jamf Pro endpoints this resource now uses report
category and site by ID alone. Where the category is managed by Terraform, read the name from that
resource. Where it is not — the usual case, since these attributes were read-only and the
assignment is commonly made in the Jamf Pro UI — look it up by the ID the title reports:

```hcl
# the category is managed by Terraform
output "category_name" {
  # value = jamfplatform_pro_patch_software_title.example.category_name
  value = jamfplatform_pro_category.example.name
}

# the category is not managed by Terraform
data "jamfplatform_pro_category" "title" {
  id = jamfplatform_pro_patch_software_title.example.category_id
}

output "category_name" {
  value = data.jamfplatform_pro_category.title.name
}
```

`jamfplatform_pro_site` covers `site_name` the same way, selected by either `id` or `name`.

State migrates automatically — the provider strips both attributes during the state upgrade, so
no `terraform state` surgery is needed. They have also been removed from the corresponding data
source.

**Grant two further permissions.** The data source, the list resource and `terraform import`
resolve a title's `source_id` from the tenant's patch source catalogues, because the endpoints
this resource now uses name its patch source without numbering it. An integration reaching this
resource through any of those three paths therefore needs **App lifecycle management → External
patch sources → Read** and **App lifecycle management → Internal patch sources → Read** alongside
**Patch titles**. Without them the read fails with `403 BAD_PERMISSIONS`, reported as `Unable to
determine the patch software title's source_id`. Planning and applying a title already in state
is unaffected, as `source_id` is carried forward rather than resolved again. The **Required Jamf
permissions** table on each of the three documentation pages lists the full set.

**`category_id` and `site_id` no longer accept `"0"`.** Use `-1`, the default, for "No category
assigned" and "NONE"; a configuration setting either attribute to `"0"` is now rejected at plan
time. This is the reverse of the retired endpoint's convention, under which `0` cleared the
assignment and `-1` was a silent no-op. A title whose state still holds `"0"` needs no state edit,
as the next refresh reads it back as `-1`; only a configuration setting the value explicitly has to
change.

Otherwise the resource behaves as before: patch software titles are now read, updated and deleted
through Jamf Pro's `/patch-software-title-configurations` endpoints rather than the deprecated
ProClassic `/patchsoftwaretitles` ones, and every attribute that remains keeps its meaning.
`version_packages` still manages only the versions it names, but it now delivers that by reading
the title's current assignments and rewriting the whole set rather than by the endpoint merging
each version in turn. An assignment created in the Jamf Pro UI between that read and the write can
therefore be lost, so avoid editing a title's **Definition** tab while an apply is in flight.

### Deprecated, not yet removed: the flat blueprint component attributes

**No action needed yet.** The flat top-level component attributes on
`jamfplatform_blueprints_blueprint`, together with the
top-level `activation_conditions` and `legacy_payloads`, are superseded by named, ordered
`component_blocks`. They remain present and functional, and may be removed on or after **22
October 2026**. Migrating ahead of that date is recommended; `terraform plan` reports every
attribute requiring migration.

### Now read-only: `unmanaged_sync_threshold` on `jamfplatform_security_cloud_uem_connect`

**Action needed if the attribute is set: remove it from your configuration.** Like
`personal_device_enrollment_type` below, this is a Jamf-side behaviour change rather than a
provider decision — but unlike that one it could not simply be retained, because it was breaking
every apply.

Jamf Security Cloud does not apply an unmanaged threshold to a Jamf Pro connection; it takes
device status from Jamf Pro directly. Wire-probed 2026-09-01: every value sent for a `JAMF_PRO`
connector is accepted and then discarded, and the connector always reports `0`. Because the
provider defaulted the attribute to `3` and sent it on every write, Terraform saw a value it had
planned come back different and failed the apply:

```
Error: Provider produced inconsistent result after apply
.unmanaged_sync_threshold: was cty.NumberIntVal(3), but now cty.NumberIntVal(0)
```

The attribute is now `Computed` and always reads `0`. Leaving it in a configuration is an error
(`Invalid Configuration for Read-Only Attribute`), so delete the line; no state edit is needed.
Nothing about how your devices are treated changes — the setting never took effect.

### Retained: `personal_device_enrollment_type`

**No action needed.** On `jamfplatform_pro_user_initiated_enrollment_settings`. This is a Jamf
Pro deprecation rather than a provider one: Jamf Pro has ignored the value since 11.25 and always
reports `USERENROLLMENT`. The attribute is read-only and is retained.

## Additions since v0.28.1

**No action needed.** The following arrived during the GA cycle, across the `v0.29.0` release
candidates. All of it is additive relative to `v0.28.1`.

### Jamf Security Cloud

A new construct family, `jamfplatform_security_cloud_*`, reached through the same gateway and the
same credentials as the rest of the provider. A Security Cloud integration may be refused with
`403 NOT_ENTITLED` despite authenticating correctly; this indicates an entitlement gap rather than
a credential problem, and the provider reports it as such.

| Area | Constructs |
|---|---|
| Custom DNS | `dns_zone` (with data sources and list resource), `dns_search_domain` and `dns_hostname_mappings` (each with a data source) |
| ZTNA gateways | `ztna_gateway` and `ztna_grouped_gateway` (both with data sources and list resources), `ztna_shared_gateways` data source |
| ZTNA access policy | `ztna_app` (with data sources and list resource), `ztna_predefined_apps` data source |
| Device groups | `security_cloud_device_group` (with data sources and list resource) — distinct from the Platform Services `jamfplatform_device_group` |
| UEM Connect | `uem_connect` (with data source and list resource), plus the `uem_connect_synchronize` and `activation_profile_deploy` actions |
| Activation profiles | `activation_profile` — a deliberately small part of a profile, with no import support and no drift detection; changing any setting replaces the profile and mints a new activation code |
| Catalogues | `content_categories` data source |

Two delete behaviours affect destroy ordering. A ZTNA gateway that is still referenced cannot be
deleted, and the provider names the referrer in the diagnostic rather than surfacing an
unqualified `409`. A Security Cloud device group referenced by a ZTNA app deletes successfully and
empties the app's assignment. The behaviour differs per construct.

Two behaviours are worth knowing before the first apply. A ZTNA gateway apply waits for the
gateway to report a settled status rather than returning as soon as Jamf accepts the write, so
creating or re-provisioning a dedicated internet gateway takes about five minutes rather than
seconds — which is what makes its `dedicated_egress_ip_addresses` usable from the same apply.
Raise `create` or `update` in the resource's `timeouts` block if a region provisions more slowly
than the default ten minutes allows. And **destroying a `uem_connect` connector in
`platform_tenant` form leaves a live Jamf Pro API integration behind**: Jamf Security Cloud creates
one named `JSC Connector` to authenticate with, nothing removes it, and the provider holds no Jamf
Pro credentials for that tenant to remove it with. The role carries 31 privileges, including
fleet-wide profile and group writes, so audit **Settings > API roles and clients** on the Jamf Pro
instance after any destroy and delete the orphans. Further detail:
[Jamf Security Cloud](security-cloud).

### Jamf Account

New in this pre-release. `jamfplatform_account_sso_domain` claims a DNS domain for your Jamf
Account organization's single sign-on, with singular and plural data sources, a list resource, and
the `jamfplatform_account_sso_domain_verify` action that re-checks the domain's DNS ownership
record. This is the provider's first *organization-scoped* family and the only one that scope
reaches: configure it by setting neither `environment_id` nor `tenant_id`, as described under
[Scope](#scope). It is served only from the US gateway.

Three things differ from every other resource in the provider, all of them wire behaviour rather
than design choices. Jamf Account exposes no read, update or patch for a single domain, so every
attribute is `RequiresReplace` and `terraform import` is **by domain name** rather than by ID —
the ID is not stable, since reclaiming a domain mints a new one. Verification is an action rather
than resource state, and it never polls: a failed check returns `200` with the status unchanged,
still moves the fourteen-day deadline, and the five-minute rate limit is measured from the moment
the domain was claimed, so the first check straight after a create is always refused. Publish the
`verification_txt_record` value — exported whole, prefix included, as the console shows it — wait
five minutes, then run the action.

### Jamf AI Governance

`jamfplatform_ai_governance_policy` manages the settings delivered to an AI tool — Claude Code,
Claude Desktop and OpenAI Codex at present — and a blueprint delivers a pinned policy version to
Macs. It ships with singular and plural data sources and a list resource.
`jamfplatform_ai_governance_tool` and `jamfplatform_ai_governance_tools` read the product
catalogue. The settings body is the tool vendor's own JSON, validated at plan time against the
schema the platform serves. This is the one construct family the provider itself gates on
environment scope, described under [Scope](#scope). Further detail:
[AI Governance policies](ai-governance-policies).

### Elsewhere

- `environment_id` and the `jamfplatform_pro_tenant_id` data source, both described under
  [Scope](#scope).
- `custom_headers` and `authorization_header_name`, for networks in which Terraform reaches Jamf
  through a proxy that authenticates callers itself. `Cookie`, `Content-Type` and `Accept` are
  refused in both, because a supplied header replaces rather than adds and would displace a value
  the provider chose per request. See
  [Reverse proxies and custom headers](reverse-proxy).
- The `jamfplatform_pro_patch_policy` list resource now names any patch policy it could not
  enumerate, in a warning on the list result, instead of dropping it silently. A
  `plan -generate-config-out` run that came back one resource short of the tenant said nothing
  about it before.
- `jamfplatform_security_cloud_uem_connect` validates `sync_refresh_interval_minutes` against the
  intervals the service actually accepts — `60`, `120`, `240`, `480`, `720` and `1440` — at plan
  time, rather than letting any other value fail the apply with an unattributed `422`.
- An incorrect `base_url` is now reported as such, rather than presenting as a network failure.
- Built against Jamf Pro 11.31.0 and Classic API 11.28.0.
- The required-permission tables throughout this documentation are regenerated against the GA
  capability model, and now name each permission as Jamf Account's permission picker does rather
  than by its retired Jamf Pro privilege name.

## Troubleshooting

| Symptom | Cause and resolution |
|---|---|
| `no schema available for <type>.<name> while reading state; this is a bug in Terraform and should be reported` | Not a Terraform defect. A removed resource remains in state; remove it with `terraform state rm`. Every operation in the workspace fails until it is removed. |
| `404 page not found`, with no JSON body | `base_url` includes a path. Supply the host only. |
| `403 OWNERSHIP_FORBIDDEN` | `environment_id` supplied for a tenant-scoped integration, or the reverse. |
| `403 BAD_PERMISSIONS` | The integration lacks a permission the construct requires; the resource's documentation page lists them. On a blueprint or compliance benchmark, also the signature of a tenant-scoped integration: those permissions are selectable only on an environment-scoped one. |
| `403 NOT_ENTITLED` on a Security Cloud construct | The tenant does not hold that Security Cloud capability. Additional permissions will not resolve it. |
| Authentication fails outright | Beta credentials in use. Register a replacement integration in Jamf Account. |
