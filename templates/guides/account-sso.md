---
page_title: "Jamf Account single sign-on"
description: |-
  Claim and verify domains, and connect an identity provider, for the organization that owns your Jamf tenants.
---

# Jamf Account single sign-on

Jamf Account is the console at [account.jamf.com](https://account.jamf.com). Single sign-on is
configured under **Organization > SSO**, and it sits above individual tenants: one connection signs
users in to Jamf Account itself and to whichever Jamf Pro, Jamf School, Jamf Protect and Jamf
Security Cloud tenants you choose.

Two things make this family different from every other one in the provider. It needs a scope no
other construct uses, and claiming a domain cannot be done in a single apply. Both are covered
below, followed by the API limits worth knowing before you build rather than after.

## What the provider manages

| In the console | Constructs |
|---|---|
| **Organization > SSO > Domains** | `jamfplatform_account_sso_domain` (resource, data source, list resource), `..._sso_domains` (data source), `..._sso_domain_verify` (action) |
| **Organization > SSO** | `jamfplatform_account_sso_connection` (resource, data source, list resource), `..._sso_connections` (data source) |

Neighbouring console pages have no constructs: users and contacts, roles and privileges, SCIM, and
agreements.

Do not confuse these with `jamfplatform_pro_sso_settings` and `jamfplatform_pro_sso_failover_url`,
which configure a single Jamf Pro tenant's own single sign-on. Different product, different
endpoint, different scope.

## The provider needs an organization-scoped integration

This is the only family the provider reaches under **Organization management** scope, and the only
scope that reaches it. A tenant-scoped integration is refused, so this is not a matter of
preference.

Create the integration in Jamf Account with the *Organization management* scope and grant it the
**SSO connections** and **SSO domains** permissions. Then configure the provider with neither
`environment_id` nor `tenant_id`:

```terraform
provider "jamfplatform" {
  base_url = "https://us.api.jamfcloud.com"
  # client_id and client_secret from the Organization management integration.
  # Set no environment_id and no tenant_id: Jamf resolves the organization from
  # the access token itself.
}
```

Setting either scope attribute breaks this family, and every other family breaks without one. So an
organization-scoped provider block cannot also manage Jamf Pro resources. Use two provider blocks
with aliases if you need both in one configuration.

Only the United States gateway serves the namespace. `base_url` must be
`https://us.api.jamfcloud.com` whatever region your tenants live in, and whatever region you choose
for a connection.

## Claiming a domain takes three applies

A connection can only name domains the organization has proved it owns, and proving ownership needs
a DNS record that does not exist until the claim is made. That sequence cannot collapse into one
apply.

### First, claim the domain and read back the record

```terraform
resource "jamfplatform_account_sso_domain" "corp" {
  domain = "example.com"
}

output "dns_record_to_publish" {
  value = jamfplatform_account_sso_domain.corp.verification_txt_record
}
```

Publish that value as a `TXT` record at the root of the domain, with a TTL of 86400. Use
`verification_txt_record` rather than assembling the record from `verification_key`: the assembled
form is what Jamf looks for, and getting the prefix wrong fails silently, by the domain simply never
verifying.

Where your DNS is managed by Terraform too, feed it straight into the relevant record resource and
this step and the publishing happen together.

### Then ask Jamf to check it

```shell
terraform apply -invoke='action.jamfplatform_account_sso_domain_verify.corp'
```

Do not wire this to the domain's own lifecycle with an `action_trigger`. Jamf allows one check every
five minutes per domain, and claiming the domain starts that clock, so a check in the same run as
the claim is refused every time and the refusal fails the apply.

Two more things about that limit. It is measured from the domain's last-modified time, and Jamf
moves that itself without any request, so a refused check means re-reading the domain and waiting
five minutes from its current `last_modified_at` rather than counting from the last attempt. And
every check moves the domain's fourteen-day verification deadline whether it succeeds or fails, so
it is not free to run repeatedly.

A check that finds nothing is reported as a failure. A green run means the domain really is
verified.

### Then add the connection

Only now will Jamf accept a connection naming that domain.

Leave the `TXT` record published permanently. Jamf re-checks in the background, and a claim lapses
fourteen days after its last successful verification.

## Changing a connection replaces it

Changing an existing connection is the one thing Jamf Account cannot currently do. Every in-place
change is refused, whatever it changes and whatever it carries, including the very body a create
accepts. Creating, reading, listing, importing and destroying connections all work. So the provider
plans a replacement for any change to any attribute.

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

A resource cannot set that for you. It is deliberately the practitioner's choice, so set it on any
connection you would mind being briefly absent. Rotating `client_secret` replaces the connection
too, since rotation is an update.

### When a create is refused with an internal failure

A refused create is reported as an internal failure that names no field, and it is the same answer
whatever the cause. Work through the causes seen so far:

- A domain in `domains` your organization has not claimed.
- A domain claimed but not yet verified.
- A required value that resolved to nothing, so it never reached Jamf.
- A settings block that disagrees with `connection_type`, such as `entra` settings on a Google
  Workspace connection.
- A `name` holding anything but letters and digits. The provider catches that one at plan time.
- The organization already holding as many connections as Jamf allows. An identical request that is
  refused at the limit succeeds once a connection is removed, so count what you already have.

## Connection names are neither free-form nor unique

Jamf accepts only letters and digits in a connection name and rejects any other character. The
provider refuses one at plan time, because Jamf's own refusal names no field.

Jamf also appends a suffix of its own to whatever name you choose, and the console hides it. Two
connections created with the same name both exist, and the console shows both under that one name.
`internal_name` and `id` are the only way to tell them apart, so give each connection a distinct
name.

## The group name filter matches on substrings

`group_name_filter` chooses which of your identity provider's groups are passed through to Jamf, for
a directory holding more groups than Jamf needs. It holds two values: an `operator`, the console's
AND/OR toggle beside the group list, where `or` passes a group matching any entry and `and` requires
every entry; and the `groups` names to match on.

The match is on substrings rather than whole names. Any group whose name *contains* one of the
strings you list is included on the token, so `groups = ["Engineering"]` also passes
`Non-Engineering-Contractors` through. List names specific enough that a partial match cannot catch
a group you did not mean.

Three states are distinct, and the block keeps them apart. Leave the whole block out and no filter
is sent at all. Supply it with an empty `groups` set and an empty filter is sent, which is a
different value and the shape most connections carry. Supply names and only the groups they match
reach the token.

## The product and tenant assignment cannot be read back

Jamf returns the product names a connection is enabled for, but never the tenants within them. So
`enabled_products` is config-authoritative: a change made in the console is invisible to
`terraform plan` and will not be corrected.

No endpoint lists an organization's tenant identifiers either, so the values for
`enabled_products[].tenants` have to come from the console or from Jamf Support.

## The callback URL is only in the console

The console displays the URL to configure in your identity provider. Jamf never returns it, so copy
it from the console rather than assembling it.

## Some connections cannot be managed

A connection created through Microsoft's admin consent flow has its client registration held by
Jamf, so the provider refuses to read or change one. It will still destroy one you had already
adopted. Domains another organization has shared with yours are likewise readable but not
manageable.

## A connection Jamf lists but cannot read

Jamf can list a connection among your organization's connections and still report it missing when
it is read on its own identifier. That is a disagreement inside Jamf between its own list and the
record behind a single connection, not a sign the connection has gone, so the provider reports it
and leaves it in state rather than planning to create it again.

A refresh can meet that state, and while it holds, a plain `terraform destroy` cannot get past it.
Run `terraform state rm` on the address, or destroy with `-refresh=false`, and raise the identifier
with Jamf Support.

## Importing a connection

Two values cannot be recovered by importing. `client_secret` is never returned, by design, and the
tenants in `enabled_products` are never returned either.

Because every change plans a replacement, a mismatch there is not a cosmetic diff. The next apply
destroys the imported connection and creates a new one, and nobody on its domains can sign in
through it while that gap is open. `client_secret` you can paste from your identity provider. The
tenants have to be written from what you know the connection covers, since nothing in Jamf will tell
you, and writing them wrong is what triggers the replacement.

So set `create_before_destroy` on any imported connection carrying real sign-in traffic, and read
the first plan after the import before applying it. A plan reporting no changes means the
configuration matches. One reporting a replacement means something does not, and applying it costs a
live connection.

## Further reading

- [Single sign-on](https://learn.jamf.com/en-US/bundle/jamf-account/page/Single_Sign-On.html) in the
  Jamf Account documentation.
- [Getting started with the Platform
  API](https://developer.jamf.com/platform-api/reference/getting-started-with-platform-api), for
  creating the integration and choosing its scope.
