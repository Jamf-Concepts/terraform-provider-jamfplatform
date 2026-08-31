---
page_title: "Jamf Security Cloud"
description: |-
  Manage Jamf Security Cloud from Terraform — ZTNA access policy, access gateways, custom DNS, device groups and the UEM Connect integration.
---

# Jamf Security Cloud

Jamf Security Cloud is the portal at [radar.jamf.com](https://radar.jamf.com) where content
controls, network security and Zero Trust Network Access are configured. This guide is the
Terraform half: which portal pages the provider reaches, what order to build in, and the
behaviours that will surprise you.

Jamf splits its own documentation across the [Portal Setup
Guide](https://learn.jamf.com/r/en-US/jamf-security-cloud-setup-guide/RADAR_Portal) (device groups,
activation profiles, integrations) and the [Zero Trust Network
Access](https://learn.jamf.com/r/en-US/jamf-connect-documentation-current/Private_Access) section
of the Jamf Connect Documentation (access policy, gateways). See [Further
reading](#further-reading).

## What the provider manages

| In the portal | Constructs |
|---|---|
| **Devices > Device groups** | `jamfplatform_security_cloud_device_group` (resource, data source, list resource), `..._device_groups` (data source) |
| **Devices > Activation profiles > Deployment** | `jamfplatform_security_cloud_activation_profile_deploy` (action) |
| **Policies > Access > Access policy** | `jamfplatform_security_cloud_ztna_app` (resource, data source, list resource), `..._ztna_apps` (data source), `..._ztna_predefined_apps` (data source), `..._content_categories` (data source) |
| **Integrations > Access gateways** | `jamfplatform_security_cloud_ztna_gateway` (resource, data source, list resource), `..._ztna_gateways` (data source), `..._ztna_grouped_gateway` (resource, data source, list resource), `..._ztna_grouped_gateways` (data source), `..._ztna_shared_gateways` (data source) |
| **Integrations > Custom DNS > DNS zones** | `jamfplatform_security_cloud_dns_zone` (resource, data source, list resource), `..._dns_zones` (data source) |
| **Integrations > Custom DNS > Search domain** | `jamfplatform_security_cloud_dns_search_domain` (resource, data source) |
| **Integrations > Custom DNS > Hostname mapping** | `jamfplatform_security_cloud_dns_hostname_mappings` (resource, data source) |
| **Integrations > UEM Connect** | `jamfplatform_security_cloud_uem_connect` (resource, data source, list resource), `..._uem_connect_synchronize` (action) |

Three neighbouring things are **not** managed here:

- **Activation profiles** are created in the portal. Most of their settings cannot be edited after
  creation, so the provider offers only the action that deploys one to Jamf Pro.
- **Content filtering and network security policies** — Jamf Protect's half of the portal — have no
  constructs yet.
- **Devices** arrive by enrolling with Jamf Trust or syncing from a UEM.

## Build it in this order

The sections below are ordered for reading. A rollout goes in this order instead:

1. **UEM Connect** — inventory and group membership have to be syncing before anything downstream
   is accurate.
2. **Device groups** — or map them from Jamf Pro groups in step 1.
3. **Gateways, then custom DNS** — a zone cannot be written before the gateway its name servers
   name. The other way round is refused with `422 GATEWAY_NOT_FOUND`, reported against
   `authoritative_name_servers`.
4. **Access policy** — apps reference the groups, categories and gateways above.
5. **Activation profile deploy** — nothing above reaches a device until this runs.

Terraform derives most of that from the references between resources. The order is yours to get
right only where there is no reference — a zone naming a gateway ID that arrived as a variable, say.

## End to end: publishing one enterprise app over ZTNA

An **Access policy** entry tells Jamf Security Cloud which traffic belongs to an application, who
may reach it, and how to route it. Two resources cover a public-facing app:

```hcl
# A device group holds nothing but a name. Membership is decided by whatever
# references the group, never on the group itself.
resource "jamfplatform_security_cloud_device_group" "engineering" {
  name = "Engineering"
}

# A gateway is named by an opaque ID, so resolve it from the catalogue. A category
# is named by its display name, so there is nothing to resolve — content_categories
# is how you confirm a spelling.
data "jamfplatform_security_cloud_content_categories" "all" {}
data "jamfplatform_security_cloud_ztna_shared_gateways" "all" {}

locals {
  nearest_data_center = one([
    for gateway in data.jamfplatform_security_cloud_ztna_shared_gateways.all.shared_gateways :
    gateway.id if gateway.name == "Nearest Data Center"
  ])
}

resource "jamfplatform_security_cloud_ztna_app" "wiki" {
  name = "Engineering Wiki"

  # Matched on display name, so the spelling has to be exact.
  category = "Business & Industry"

  # A wildcard may replace the whole leading label, and entries must be mutually
  # exclusive — "*.wiki.example.com" does not cover "wiki.example.com", so the
  # parent is listed separately.
  hostnames = ["wiki.example.com", "*.wiki.example.com"]

  all_device_groups = false
  device_group_ids  = [jamfplatform_security_cloud_device_group.engineering.id]

  routing = {
    traffic_routing = "Encrypt and route via ZTNA"
    routing_mode    = "Standard"
    gateway_id      = local.nearest_data_center
  }

  # A Security card left out is one Jamf keeps its own setting for. Set
  # enabled = false to lift a requirement; removing the block only stops
  # Terraform managing it.
  security = {
    managed_device = {
      enabled                   = true
      device_push_notifications = true
    }
    device_risk = {
      enabled            = true
      deny_at_risk_level = "High"
    }
    jamf_trust = {
      enabled = true
    }
  }
}
```

Two of Jamf's rules govern that `security` block. It **needs a Jamf Protect licence** — these are
the portal's "Access requires device to be managed / device risk validation / Jamf Trust to be
enabled" cards. And the managed-device requirement is only as accurate as UEM Connect: a device
counts as managed because [the sync](#uem-connect) says so.

### Predefined applications

An application is either **custom**, as above, or **predefined** — built from one of Jamf's
definitions, which brings its own host names:

```hcl
data "jamfplatform_security_cloud_ztna_predefined_apps" "all" {}

resource "jamfplatform_security_cloud_ztna_app" "github" {
  predefined_app_id = one([
    for app in data.jamfplatform_security_cloud_ztna_predefined_apps.all.predefined_apps :
    app.id if app.name == "GitHub"
  ])

  category          = "Communication"
  all_device_groups = true

  routing = {
    traffic_routing = "Encrypt and route via ZTNA"
    routing_mode    = "Standard"

    # Declared in the example above.
    gateway_id = local.nearest_data_center
  }
}
```

A predefined application takes its name from the definition, so `name` is not accepted; `hostnames`
are *added to* the definition's rather than replacing them; and only one application per definition
is allowed on a tenant.

**One line decides which form you need: a predefined application requires every host name to be
publicly resolvable.** Names that resolve only on your own network — split-brain DNS — have to be a
custom application paired with a custom DNS zone. See [Adding a New Predefined
Application](https://learn.jamf.com/r/en-US/jamf-connect-documentation-current/Adding_a_New_SaaS_Application).

## Reaching private applications: custom DNS

An app on an internal network is not reachable just because an access policy names it — its host
names have to resolve, and public DNS will not resolve them. That needs a **custom DNS zone**
naming your own authoritative name servers, and a gateway those name servers are reachable through.

```hcl
resource "jamfplatform_security_cloud_dns_zone" "internal" {
  name = "Corp internal"

  # A parent domain and its subdomains are separate entries: "example.corp"
  # covers only the parent, "*.example.corp" only the subdomains.
  domains = ["example.corp", "*.example.corp"]

  # local.nearest_data_center comes from the example above and is the shared
  # egress. A name server on your own network is not reachable through it and
  # needs a dedicated gateway.
  authoritative_name_servers = [
    { ip_address = "203.0.113.53", gateway_id = local.nearest_data_center },
    { ip_address = "203.0.113.54", gateway_id = local.nearest_data_center },
  ]
}
```

Queries matching the zone's domains go to one of its name servers by pseudo-random load balancing.
Four of Jamf's rules each produce a plan that applies and then fails to work:

- **A domain belongs to exactly one zone**, tenant-wide, and an IP address may appear only once per
  zone.
- **Reserved addresses are refused** — a private or loopback name server fails with
  `Name server IP address not allowed`. Unrouted is not the same as reserved: a documentation range
  such as `203.0.113.0/24` is accepted, which is why the example uses one.
- **A dedicated *internet* gateway refuses a private address too**, though Jamf's documentation
  attaches the restriction only to the shared egress. Reaching a name server on an internal address
  means the IPsec form below.
- **Reverse lookups** work by adding the `.in-addr.arpa` and `.ipv6.arpa` domains, and **changes to
  your own infrastructure need the zone changed too** — a renumbered name server, or an app on a
  domain the zone does not match.

Misconfiguring a zone cuts users off from everything behind it. Jamf's [Configuring Custom DNS
Zones](https://learn.jamf.com/r/en-US/jamf-security-cloud-setup-guide/Configuring_Custom_DNS_Zones)
and [Troubleshooting Custom DNS
Zones](https://learn.jamf.com/r/en-US/jamf-security-cloud-setup-guide/Troubleshooting_Custom_DNS_Zones)
are the pages to read before and after.

### Gateways

Three kinds can be named as a zone's `gateway_id` or an app's routing target, and only two are
yours to create:

- **Shared** — Jamf-operated, available to every entitled tenant: "Nearest Data Center", which
  routes through whichever data centre is closest to the device, plus a set of regional shared IP
  pools. Read them with `..._ztna_shared_gateways`; they cannot be modified, deleted, or made a
  member of a group.
- **Dedicated** — your own egress point, and a paid add-on. An `ipsec` block builds a tunnel to your
  VPN concentrator; omitting it provisions a pair of private egress IPs, reported in
  `dedicated_egress_ip_addresses`.
- **Grouped** — a failover and routing group over two or more dedicated gateways of the *same* form,
  all IPsec or all internet. Shared gateways are refused as members.

```hcl
resource "jamfplatform_security_cloud_ztna_gateway" "frankfurt" {
  name          = "Frankfurt DC"
  egress_region = "Europe - Germany"
  tenant_ids    = [var.security_cloud_tenant_id]

  contact = {
    name  = "Network Operations"
    email = "netops@example.com"
  }

  # Omitted, so this is a dedicated internet gateway. The full IPsec form, with
  # its jamf_side and customer_side blocks, is on this resource's own page.
  # ipsec = { ... }
}
```

`tenant_ids` is mandatory, and every tenant in it must belong to the same organization as the
provider's credentials. No API lists an environment's tenants, so an environment-scoped
configuration cannot fill it in from data — supply it as an input, or configure the provider with
`tenant_id`.

**A completed apply is not a usable gateway.** A new one reports `PENDING` and settles to `UP`
roughly four and a half minutes later. The egress addresses arrive far sooner — within seconds — so
`dedicated_egress_ip_addresses` is populated long before the gateway carries any traffic.

Changing `egress_region` re-provisions it: connectivity drops and the status returns to `PENDING`.
For about half a minute the attribute still holds the **old** region's addresses, then the new ones
replace them in place — it never empties, so a value read during that window is plausible and wrong.
Both timings come from a single EU measurement; treat them as orders of magnitude.

### Short names and fixed addresses

Two more **Custom DNS** settings, both one-per-tenant:

```hcl
# Completes a short host name: a user asking for "wiki" is directed to
# "wiki.example.corp". Setting this is also what lets an access policy's
# hostnames be written as short names.
resource "jamfplatform_security_cloud_dns_search_domain" "corp" {
  domain_name = "example.corp"
}

# Fixed answers for internal host names. This resource owns the tenant's ENTIRE
# set: a mapping added elsewhere and absent here is removed on the next apply.
resource "jamfplatform_security_cloud_dns_hostname_mappings" "internal" {
  mappings = [
    {
      hostname       = "build.example.corp"
      ipv4_addresses = ["10.10.5.20"]

      # Both are required on every mapping. The admin UI's add dialog
      # pre-selects Connect to ZTNA, so a mapping added there and one written
      # here do not start from the same value.
      connect_to_ztna       = true
      connect_to_secure_dns = false
    },
    {
      hostname              = "artifacts.example.corp"
      ipv4_addresses        = ["10.10.5.21"]
      connect_to_ztna       = true
      connect_to_secure_dns = false
    },
  ]
}
```

Between 1 and 500 mappings, each host name once, and at least one of `ipv4_addresses` or
`ipv6_addresses` per entry. `mappings = []` is not accepted — destroy the resource to remove them
all. Destroying the search domain clears it for the tenant. Declare each of these once: a second
instance of either will fight itself on every apply.

## Traffic matching is tenant-unique

A host name or address range belongs to **one** application across the whole tenant, and claiming
one another application holds is rejected at apply. **That makes rename-by-replacement fail:**
Terraform creates the new application before destroying the old one, so the create hits the old
one's names. Split it across two applies, or move the resource with `terraform state` instead of
replacing it.

Within a tenant Jamf resolves overlaps by specificity — the most granular host name wins, so
`images.app.example.com` beats `*.example.com`, and a custom application's host name beats the same
name inside a predefined definition. That is what makes Jamf's recommended pattern work: start with
one wildcard application to get traffic flowing, then carve specific applications out of it without
touching the wildcard. See the [Network Engineer's Guide to Jamf Connect
ZTNA](https://trusted.jamf.com/docs/network-engineers-guide-to-private-access).

`direct_ips_and_subnets` carries two further limits, both Jamf's: it needs the Jamf Trust app on the
device, and direct-IP traffic does not appear in the access reports. Prefer host names where the
application supports them.

## Delete ordering

Jamf enforces references on delete three different ways:

| Destroying | What happens |
|---|---|
| A **gateway** a DNS zone, grouped gateway or app still references | Refused, naming what holds it. |
| A **grouped gateway's member** while it is still in the group | Refused. |
| A **device group** an app assignment or UEM mapping still names | **Succeeds silently**, dropping the group from every assignment that named it — which can leave an application assigned to nobody. |

**The first carries a Terraform trap on top of the refusal.** Dropping the reference *and* the
gateway in one apply does not work either, because Terraform sequences the destroy before the
update that would have released it. Release in one apply, destroy in the next.

**The third is the dangerous one, because nothing fails** — check what references a device group
before removing it. The built-in **Default Group** cannot be managed here at all: Jamf gives it no
identifier and reserves the name. It appears in `..._device_groups` with `built_in = true` and a
null `id`.

## Immutable forms

Three constructs pick a shape at create time that Jamf will not convert, so Terraform replaces the
object:

- A **gateway** is IPsec or internet, decided by whether `ipsec` is present.
- A **ZTNA app** is predefined or custom, decided by whether `predefined_app_id` is set.
- A **UEM Connect** integration's vendor, address and authentication method are fixed. Changing any
  of them replaces the integration, briefly interrupting syncing.

## UEM Connect

UEM Connect syncs device inventory and group membership from Jamf Pro into Jamf Security Cloud and
signals device risk back. It is what makes the managed-device requirement mean anything, and what
lets you keep deciding group membership in Jamf Pro.

```hcl
# The Jamf Pro tenant identifier is the one value here you cannot read off the
# UEM Connect screen, so read it rather than copying it between consoles.
data "jamfplatform_pro_tenant_id" "jamf_pro" {}

resource "jamfplatform_security_cloud_uem_connect" "jamf_pro" {
  uem_vendor = "JAMF_PRO"
  enabled    = true

  # Prefer platform_tenant: Jamf Security Cloud creates and manages its own
  # credentials on the named tenant, so no secret is configured here. The
  # alternative, oauth, takes the client ID and secret of an API integration you
  # created on the Jamf Pro instance yourself, plus uem_server_url.
  platform_tenant = {
    tenant_id = data.jamfplatform_pro_tenant_id.jamf_pro.tenant_id
  }

  group_membership_mapping = {
    enabled = true

    # Evaluated in order: a device joins the group of the first entry it
    # matches, so put the most specific first. A device matching nothing joins
    # the default group.
    mappings = [
      {
        # A Jamf Pro group is computer_ or mobile_ followed by its number, which
        # composes out of a jamfplatform_device_group:
        #   "${jamfplatform_device_group.x.device_type}_${jamfplatform_device_group.x.jamf_pro_id}"
        uem_group_id            = "computer_12"
        security_cloud_group_id = jamfplatform_security_cloud_device_group.engineering.id
      },
    ]
  }
}

# Starts a sync immediately rather than waiting for the scheduled one. Jamf runs
# it in the background, so this returns once the run has started and reports
# nothing about what it did — read latest_sync from the data source afterwards.
action "jamfplatform_security_cloud_uem_connect_synchronize" "now" {
  config {
    uem_connect_id = jamfplatform_security_cloud_uem_connect.jamf_pro.id
  }
}
```

`jamfplatform_pro_tenant_id` reaches the Jamf Pro namespace, so it works under a platform
environment — resolving the Jamf Pro tenant in that environment — and under tenant scope pointed at
the Jamf Pro tenant. It does **not** work under tenant scope pointed at the Security Cloud tenant,
where Jamf Pro does not answer; supply the identifier as an input there. This is not the same
identifier as a gateway's `tenant_ids`, which names Security Cloud tenants and has no data source
to read.

Jamf checks neither side of a mapping, so a wrong group number is accepted and simply never
matches. **The group configuration is replaced wholesale on every apply, so there is no way to leave
it unmanaged.** Declaring `group_membership_mapping` replaces what it does not mention — an omitted
or empty `mappings` clears every mapping — and omitting the block entirely resets the whole group
configuration to its defaults. Manage it, or expect it cleared.

A tenant holds one integration and a second create is refused, so **import** where one already
exists. The ID is not a value anyone would have written down: read it off the data source
(`data "jamfplatform_security_cloud_uem_connect" "existing" {}` reports it as `id`), or let
`terraform query -generate-config-out=uem_connect.tf` write the import block from the list resource.

```shell
terraform import jamfplatform_security_cloud_uem_connect.jamf_pro 6a91b958619ef153a5a63d72
```

**Import has one wrinkle worth knowing in advance.** `user_data_field_mapping` and
`group_membership_mapping` are captured from the tenant even though your configuration may not
declare them. Run `terraform plan` straight afterwards and write in what it shows, rather than
letting the next apply clear them.

## Getting the configuration onto devices

Nothing above reaches a device on its own. Devices enrol through an **activation profile**
distributed alongside the Jamf Trust app, and the provider covers one step of that: the portal's
**Deploy to Jamf Pro** button, which creates the activation profile's configuration profile in Jamf
Pro and scopes it.

```hcl
action "jamfplatform_security_cloud_activation_profile_deploy" "macos" {
  config {
    activation_profile_code = var.activation_profile_code
    os                      = "macos"
    jamf_pro_group_ids      = ["1", "2"]
  }
}
```

- The **activation profile code** is issued during activation profile setup and cannot be created or
  looked up from Terraform. It is the last path segment of the profile's deployment page, so it is
  an input.
- There is **one deployment per operating system** (`macos`, `ios_byod`, `ios_supervised`,
  `ios_unsupervised`). Cover more than one by invoking the action once each, naming computer groups
  for `macos` and mobile device groups otherwise.
- **Scope only accumulates.** `jamf_pro_group_ids` adds groups and never removes one, and omitting
  it on a *first* deployment scopes the profile to nothing while still reporting success. Narrow or
  clear a scope in Jamf Pro.

## Requirements

Jamf Security Cloud is reached under **either** scope: set `environment_id` (preferred) or
`tenant_id` on the provider. There is no version gate — it is continuously deployed, and a tenant
can hold it without holding Jamf Pro.

Two things about authorisation are specific to this namespace:

- **Entitlement is not authentication.** A valid API integration can still be refused with
  `403 NOT_ENTITLED` when the tenant does not hold the product behind the construct — dedicated
  gateways, for instance, are a paid add-on. Every resource and action translates that into a named
  diagnostic; the three read-only catalogues (`..._content_categories`, `..._ztna_predefined_apps`,
  `..._ztna_shared_gateways`) surface the raw error instead.
- **Permissions are per construct**, granted in Jamf Account's permission picker. Every resource,
  data source and action page carries its own table naming the category, row and boxes to tick. The
  capabilities across this namespace are `ztna`, `device-groups`, `content-categories`,
  `custom-hostname-mappings`, `search-domains` and `uem-connect`.

## Further reading

| Topic | Page |
|---|---|
| The portal, end to end, in Jamf's words | [Portal Setup Guide](https://learn.jamf.com/r/en-US/jamf-security-cloud-setup-guide/RADAR_Portal) |
| What an access policy is made of | [Access Policy](https://learn.jamf.com/r/en-US/jamf-connect-documentation-current/Access_Policies) |
| Predefined and custom applications | [Predefined](https://learn.jamf.com/r/en-US/jamf-connect-documentation-current/Adding_a_New_SaaS_Application), [Custom](https://learn.jamf.com/r/en-US/jamf-connect-documentation-current/Adding_a_New_Enterprise_Application) |
| Gateway types, grouping and routing strategies | [Shared Internet Gateways](https://learn.jamf.com/r/en-US/jamf-connect-documentation-current/Internet_Cloud_Gateways), [Grouped Gateways](https://learn.jamf.com/r/en-US/jamf-connect-documentation-current/Grouped_Gateways), [Creating a Group of Gateways](https://learn.jamf.com/r/en-US/jamf-connect-documentation-current/Creating_a_Grouped_Gateway) |
| Custom DNS zones | [DNS Zones](https://learn.jamf.com/r/en-US/jamf-security-cloud-setup-guide/Custom_DNS_Zones), [Configuring](https://learn.jamf.com/r/en-US/jamf-security-cloud-setup-guide/Configuring_Custom_DNS_Zones), [Troubleshooting](https://learn.jamf.com/r/en-US/jamf-security-cloud-setup-guide/Troubleshooting_Custom_DNS_Zones) |
| UEM Connect, and its settings | [Overview](https://learn.jamf.com/r/en-US/jamf-security-cloud-setup-guide/UEM_Connect_Overview), [Settings Reference](https://learn.jamf.com/r/en-US/jamf-security-cloud-setup-guide/UEM_Connect_Settings), [Integration by Vendor](https://learn.jamf.com/r/en-US/jamf-security-cloud-setup-guide/UEM_Vendor_Integration) |
| Activation profiles | [Activation Profiles](https://learn.jamf.com/r/en-US/jamf-security-cloud-setup-guide/Activation_Profiles), [Creating one](https://learn.jamf.com/r/en-US/jamf-security-cloud-setup-guide/Creating_Activation_Profiles) |
| The policies this provider does not manage yet | [Policies](https://learn.jamf.com/r/en-US/jamf-security-cloud-setup-guide/Policies) |
| How the routing fabric actually works | [Network Engineer's Guide to Jamf Connect ZTNA](https://trusted.jamf.com/docs/network-engineers-guide-to-private-access) |
