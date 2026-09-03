---
page_title: "Jamf Account single sign-on"
description: |-
  Claim and verify domains, and connect an identity provider, for the organization that owns your Jamf tenants.
---

# Jamf Account single sign-on

Jamf Account is the portal at [account.jamf.com](https://account.jamf.com). Single sign-on is
configured under **Organization > SSO**, and it sits above individual tenants: one connection signs
users in to Jamf Account itself and to whichever Jamf Pro, Jamf School, Jamf Protect and Jamf
Security Cloud tenants you choose.

This guide covers the two things that make this family different from every other one in the
provider — a scope that no other construct uses, and a claim-then-prove sequence that cannot happen
in a single apply — followed by the API limits worth knowing before you build.

## What the provider manages

| In the portal | Constructs |
|---|---|
| **Organization > SSO > Domains** | `jamfplatform_account_sso_domain` (resource, data source, list resource), `..._sso_domains` (data source), `..._sso_domain_verify` (action) |
| **Organization > SSO** | `jamfplatform_account_sso_connection` (resource, data source, list resource), `..._sso_connections` (data source) |

Neighbouring portal pages — users and contacts, roles and privileges, SCIM, agreements — have no
constructs.

Do not confuse these with `jamfplatform_pro_sso_settings` and
`jamfplatform_pro_sso_failover_url`, which configure a **single Jamf Pro tenant's** own single
sign-on. Different product, different endpoint, different scope.

## The provider needs an organization-scoped integration

This is the only family the provider reaches under **Organization management** scope, and the only
scope that reaches it. A tenant-scoped integration is refused, so this is not a matter of
preference.

Create the integration in Jamf Account with the *Organization management* scope, grant it the
**SSO connections** and **SSO domains** permissions, then configure the provider with **neither**
`environment_id` nor `tenant_id`:

```terraform
provider "jamfplatform" {
  base_url = "https://us.api.jamfcloud.com"
  # client_id and client_secret from the Organization management integration.
  # Set no environment_id and no tenant_id: Jamf resolves the organization from
  # the access token itself.
}
```

Two consequences:

- **Setting either scope attribute breaks this family**, and every other family breaks without one.
  So an organization-scoped provider block cannot also manage Jamf Pro resources. Use two provider
  blocks with aliases if you need both in one configuration.
- **Only the United States gateway serves it.** `base_url` must be
  `https://us.api.jamfcloud.com` regardless of where your tenants live, and regardless of the
  region you choose for a connection.

## Claiming a domain takes three applies

A connection can only name domains the organization has proved it owns, and proving ownership needs
a DNS record that does not exist until the claim is made. That sequence cannot collapse into one
apply, so plan for three:

**One — claim the domain, and read back the record to publish.**

```terraform
resource "jamfplatform_account_sso_domain" "corp" {
  domain = "example.com"
}

output "dns_record_to_publish" {
  value = jamfplatform_account_sso_domain.corp.verification_txt_record
}
```

Publish that value as a `TXT` record at the root of the domain, with a TTL of 86400. Use
`verification_txt_record` rather than assembling the record from `verification_key` — the assembled
form is what Jamf looks for, and getting the prefix wrong fails silently, by the domain simply never
verifying.

Where your DNS is managed by Terraform too, feed it straight into the relevant record resource and
this step and the publishing happen together.

**Two — ask Jamf to check it.**

```shell
terraform apply -invoke='action.jamfplatform_account_sso_domain_verify.corp'
```

Do **not** wire this to the domain's own lifecycle with an `action_trigger`. Jamf allows one check
every five minutes per domain and claiming the domain starts that clock, so a check in the same run
as the claim is refused every time, and the refusal fails the apply.

Two more things about that limit. It is measured from the domain's last-modified time, and Jamf
moves that itself without any request — so if a check is refused, re-read the domain and wait five
minutes from its *current* `last_modified_at` rather than counting from when you last tried. And
every check moves the domain's fourteen-day verification deadline whether it succeeds or fails, so
it is not free to run repeatedly.

A check that finds nothing is reported as a failure, so a green run means the domain really is
verified.

**Three — add the connection.** Only now will Jamf accept one naming that domain.

Leave the `TXT` record published permanently. Jamf re-checks in the background, and a claim lapses
fourteen days after its last successful verification.

## Changing a connection replaces it

Jamf Account currently has no working update endpoint for connections: every in-place change is
refused. So the provider plans a **replacement** for any change to any attribute.

That matters for a connection carrying real sign-in traffic. Terraform destroys before it creates by
default, so users on the connection's domains cannot authenticate through it for the width of that
gap. Jamf allows two connections on one domain, so closing the gap is one lifecycle block:

```terraform
resource "jamfplatform_account_sso_connection" "corp" {
  # ...

  lifecycle {
    create_before_destroy = true
  }
}
```

A resource cannot set that for you — it is deliberately the practitioner's choice — so set it on any
connection you would mind being briefly absent.

This also means **rotating `client_secret` replaces the connection**, since rotation is an update.

## Limits worth knowing before you build

**Connection names must be letters and digits, and are not unique.** Jamf rejects any other
character, and the provider refuses it at plan time because Jamf's own refusal names no field. Jamf
also appends a suffix of its own to whatever name you choose and the portal hides it — so two
connections created with the same name both exist and the portal shows both under that one name.
`internal_name` and `id` are the only way to tell them apart. Give each connection a distinct name.

**The product and tenant assignment cannot be read back.** Jamf returns the product names a
connection is enabled for but never the tenants within them, so `enabled_products` is
config-authoritative: a change made in the portal is invisible to `terraform plan` and will not be
corrected. There is also no endpoint that lists an organization's tenant identifiers, so the values
for `enabled_products[].tenants` have to come from the portal or from Jamf support.

**The callback URL is not available here.** The portal displays the URL to configure in your
identity provider; the API does not return it. Copy it from the portal rather than assembling it.

**Some connections cannot be managed at all.** A connection created through Microsoft's admin
consent flow has its client registration held by Jamf, so the provider refuses to read or change one
— though it will still destroy one you had already adopted. Domains another organization has shared
with yours are likewise readable but not manageable.

**Importing a connection loses two things.** `client_secret` is never returned by design, and
`enabled_products` cannot be read back, so set both to match after importing or the next plan shows
a change.

## Further reading

- [Single sign-on](https://learn.jamf.com/en-US/bundle/jamf-account/page/Single_Sign-On.html) in the
  Jamf Account documentation.
- [Getting started with the Platform
  API](https://developer.jamf.com/platform-api/reference/getting-started-with-platform-api), for
  creating the integration and choosing its scope.
