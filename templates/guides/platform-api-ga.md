---
page_title: "Preparing for the Platform API GA"
description: |-
  What changes when the Jamf Platform API leaves beta: new credentials, a new base URL, environment scope, and the constructs that were removed.
---

# Preparing for the Platform API GA

The Jamf Platform API is leaving public beta and will reach general availability shortly. The
date will be announced separately.

**Action is needed in every configuration built against the public beta.** Nothing carries over
untouched: the gateway host, the credentials and the scope attribute all change, and several
constructs have been removed. Upgrade promptly once GA is announced, as the beta gateway is
retired at that point and all provider versions prior to `v0.29.0-rc.3` are bound to it.

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
      version = "0.29.0-rc.3"
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

This release can be adopted before GA. Setting `base_url` to the GA gateway is sufficient: an
existing public-beta API integration authenticates against that host, and beta credentials remain
valid until GA. Adopting early leaves only the credential replacement outstanding at GA, with the
remaining changes already validated against your own configuration.

## Action needed

In summary, per workspace. The first group applies whenever this release is adopted, including
during the public beta; the second only at GA.

**On upgrading to this release:**

1. **Remove the deleted resources from state.** After the upgrade, a workspace still holding one of
   them cannot produce a plan of any kind. See [Removed constructs](#removed-constructs).
2. **Set `base_url` to the GA gateway in the same change as the provider upgrade.** The host and
   the provider version are paired. See [Base URL](#base-url).
3. Take a state backup, as for any breaking provider upgrade.

Existing beta credentials and `tenant_id` continue to work through this upgrade. Nothing else is
required to adopt the release before GA.

**At GA:**

4. **Register a replacement API integration and update the provider credentials.** Record the
   permissions held by each beta integration beforehand, so the equivalents can be selected. See
   [API integration credentials](#api-integration-credentials).
5. **Replace `tenant_id` with `environment_id`**, if the replacement integration is
   environment-scoped, as most will be. See [Scope](#scope).

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
names take the form `compliance-benchmarks:create` or `device-groups:read`. Every resource, data
source, list resource and action page in this documentation carries a **Required Jamf privileges**
table already expressed in that form. Use those tables to select permissions for the replacement
integration, granting only what the constructs in use require.

**Three scope levels are available.** A *platform environment* is a group of tenants across
product types, and is the only scope with access to the Platform APIs. A *tenant* scope targets a
single Jamf Pro, Jamf School, Jamf Protect or Jamf Security Cloud tenant, for single-product
access. An *organization management* scope reaches a first set of organization-level resources;
the provider currently rejects it at configure time with an explanatory diagnostic, as none of the
constructs requiring it have been built. The scope selected determines the provider configuration
described under [Scope](#scope).

## Base URL

**Action needed: change the host.** The gateway host and the provider version are paired in both
directions:

| Provider version | `base_url` |
|---|---|
| `v0.29.0-rc.2` and earlier | `https://{region}.apigw.jamf.com` — the beta host. The GA host is not supported on these versions. |
| `v0.29.0-rc.3` and later | `https://{region}.api.jamfcloud.com` — required. The beta host is retired at GA. |

Change the host and upgrade the provider in a single change. Neither the GA host on an earlier
version nor the beta host on `v0.29.0-rc.3` or later is a supported combination. The GA host is
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
integrations without a platform environment. A tenant-scoped GA integration remains valid where
single-product access is intended, but does not reach the Platform API constructs, the AI
Governance policies among them.

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

### Deprecated, not yet removed: the flat blueprint component attributes

**No action needed yet.** The flat top-level component attributes on
`jamfplatform_blueprints_blueprint`, together with the
top-level `activation_conditions` and `legacy_payloads`, are superseded by named, ordered
`component_blocks`. They remain present and functional, and may be removed on or after **22
October 2026**. Migrating ahead of that date is recommended; `terraform plan` reports every
attribute requiring migration.

### Retained: `personal_device_enrollment_type`

**No action needed.** On `jamfplatform_pro_user_initiated_enrollment_settings`. This is a Jamf
Pro deprecation rather than a provider one: Jamf Pro has ignored the value since 11.25 and always
reports `USERENROLLMENT`. The attribute is read-only and is retained.

## Additions since v0.28.1

**No action needed.** The following arrived during the GA cycle, across `v0.29.0-rc.1`,
`v0.29.0-rc.2` and this pre-release. All of it is additive.

### Jamf Security Cloud

A new construct family, `jamfplatform_security_cloud_*`, reached through the same gateway and the
same credentials as the rest of the provider. A Security Cloud integration may be refused with
`403 NOT_ENTITLED` despite authenticating correctly; this indicates an entitlement gap rather than
a credential problem, and the provider reports it as such.

| Area | Constructs |
|---|---|
| Custom DNS | `dns_zone` (with data sources and list resource), `dns_search_domain`, `dns_hostname_mappings` |
| ZTNA gateways | `ztna_gateway` and `ztna_grouped_gateway` (both with data sources and list resources), `ztna_shared_gateways` data source |
| ZTNA access policy | `ztna_app` (with data sources and list resource), `ztna_predefined_apps` data source |
| Device groups | `security_cloud_device_group` (with data sources and list resource) |
| UEM Connect | `uem_connect` (with data source and list resource), plus the `uem_connect_synchronize` and `activation_profile_deploy` actions |
| Catalogues | `content_categories` data source |

Two delete behaviours affect destroy ordering. A ZTNA gateway that is still referenced cannot be
deleted, and the provider names the referrer in the diagnostic rather than surfacing an
unqualified `409`. A Security Cloud device group referenced by a ZTNA app deletes successfully and
empties the app's assignment. The behaviour differs per construct.

### Jamf AI Governance

`jamfplatform_ai_governance_policy` manages the settings delivered to an AI tool — Claude Code,
Claude Desktop and OpenAI Codex at present — and a blueprint delivers a pinned policy version to
Macs. `jamfplatform_ai_governance_tool` and `jamfplatform_ai_governance_tools` read the product
catalogue. The settings body is the tool vendor's own JSON, validated at plan time against the
schema the platform serves. Further detail:
[AI Governance policies](ai-governance-policies).

### Elsewhere

- `environment_id` and the `jamfplatform_pro_tenant_id` data source, both described under
  [Scope](#scope).
- `custom_headers` and `authorization_header_name`, for networks in which Terraform reaches Jamf
  through a proxy that authenticates callers itself. See
  [Reverse proxies and custom headers](reverse-proxy).
- An incorrect `base_url` is now reported as such, rather than presenting as a network failure.
- Built against Jamf Pro 11.31.0 and Classic API 11.28.0.
- The required-privilege tables throughout this documentation are regenerated in the GA
  `capability:action` form.

## Troubleshooting

| Symptom | Cause and resolution |
|---|---|
| `no schema available for <type>.<name> while reading state; this is a bug in Terraform and should be reported` | Not a Terraform defect. A removed resource remains in state; remove it with `terraform state rm`. Every operation in the workspace fails until it is removed. |
| `404 page not found`, with no JSON body | `base_url` includes a path. Supply the host only. |
| `403 OWNERSHIP_FORBIDDEN` | `environment_id` supplied for a tenant-scoped integration, or the reverse. |
| `403 BAD_PERMISSIONS` | The integration lacks a permission the construct requires. The resource's documentation page lists them. |
| `403 NOT_ENTITLED` on a Security Cloud construct | The tenant does not hold that Security Cloud capability. Additional permissions will not resolve it. |
| Authentication fails outright | Beta credentials in use. Register a replacement integration in Jamf Account. |
