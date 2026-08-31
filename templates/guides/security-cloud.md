---
page_title: "Jamf Security Cloud"
description: |-
  Manage Jamf Security Cloud from Terraform — ZTNA access policy, access gateways, custom DNS, device groups and the UEM Connect integration.
---

# Jamf Security Cloud

Jamf Security Cloud is the portal at [radar.jamf.com](https://radar.jamf.com) where content controls, network security and Zero Trust Network Access are configured. Jamf's own documentation is split across two guides, and both are worth having open: the [Jamf Security Cloud Portal Setup Guide](https://learn.jamf.com/r/en-US/jamf-security-cloud-setup-guide/RADAR_Portal) covers device groups, activation profiles and the portal's integrations, and the [Zero Trust Network Access](https://learn.jamf.com/r/en-US/jamf-connect-documentation-current/Private_Access) section of the Jamf Connect Documentation covers access policy and gateways. See [Further reading](#further-reading) below.

This guide is the Terraform half: which portal pages the provider reaches, the order things have to be created in, and the handful of behaviours that will surprise you.

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

Three things next door to this are **not** managed here. **Activation profiles** themselves are created in the portal — most of their settings cannot be edited after creation, so Jamf treats them as immutable and the provider offers only the action that deploys one to Jamf Pro. **Content filtering and network security policies** (Jamf Protect's half of the portal) have no constructs yet. And **devices** arrive by enrolling with Jamf Trust or by syncing from a UEM; nothing here enrols one.

## Build it in this order

The sections below are ordered for reading. A rollout goes in a different order, because each step depends on the one above it:

1. **UEM Connect** — device inventory and group membership have to be syncing before anything downstream is accurate.
2. **Device groups** — or map them from Jamf Pro groups in step 1.
3. **Gateways, then custom DNS** — a zone cannot be written before the gateway its name servers name. Written the other way round, Jamf Security Cloud refuses it with `422 GATEWAY_NOT_FOUND`, and the provider reports that against `authoritative_name_servers` rather than against the zone as a whole.
4. **Access policy** — applications reference the groups, categories and gateways above.
5. **Activation profile deploy** — nothing above reaches a device until this runs.

Terraform derives most of that from the references between resources, so the order is yours to get right only where there is no reference to derive it from — a zone naming a gateway ID that arrived as a variable, say.

## End to end: publishing one enterprise app over ZTNA

An entry on the **Access policy** page is what makes Jamf Security Cloud recognise traffic as belonging to an application, decide who may reach it, and route it. Two resources are enough for a public-facing app:

```hcl
# Who may reach it. A device group holds nothing but a name — membership is decided
# by whatever references the group, never on the group itself.
resource "jamfplatform_security_cloud_device_group" "engineering" {
  name = "Engineering"
}

# A gateway is named by an opaque ID, so resolve it from the catalogue rather than
# hard-coding one. A category is named by its display name, so there is nothing to
# resolve — the content_categories data source is how you confirm a spelling.
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

  # Matched on the category's display name, so the spelling has to be exact —
  # check it against the content_categories data source above.
  category = "Business & Industry"

  # Traffic matching. A wildcard may replace the whole leading label, and entries
  # must be mutually exclusive — "*.wiki.example.com" already covers a subdomain
  # under it, so the parent domain is listed separately.
  hostnames = ["wiki.example.com", "*.wiki.example.com"]

  # Device group permissions.
  all_device_groups = false
  device_group_ids  = [jamfplatform_security_cloud_device_group.engineering.id]

  # Application traffic routing.
  routing = {
    traffic_routing = "Encrypt and route via ZTNA"
    routing_mode    = "Standard"
    gateway_id      = local.nearest_data_center
  }

  # The Security tab. A block left out is one Jamf Security Cloud keeps its own
  # setting for; set enabled = false to lift a requirement rather than removing
  # the block, which only stops Terraform managing it.
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

Two notes on that configuration that come from Jamf's own documentation rather than from Terraform. The **security requirements need a Jamf Protect licence** — they are the portal's "Access requires device to be managed / device risk validation / Jamf Trust to be enabled" cards. And the managed-device requirement is only as accurate as UEM Connect: a device counts as managed because the sync says so, which is one reason [UEM Connect](#uem-connect) comes before access policy in a real rollout.

### Predefined applications

An application is either **custom**, as above, or **predefined** — based on one of Jamf's own definitions, which brings its own host names with it:

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

    # Declared in the end-to-end example above.
    gateway_id = local.nearest_data_center
  }
}
```

A predefined application takes its name from the definition, so `name` is not accepted; `hostnames` become *additions* to the definition's own rather than a replacement for them; and only one application per definition is allowed on a tenant. The choice between the two forms is fixed — changing `predefined_app_id` in either direction replaces the application.

Jamf draws one further line here that decides which form you need: **a predefined application requires every host name to be publicly resolvable**. An application whose names resolve only on your own network — split-brain DNS — has to be a custom application, paired with a custom DNS zone below. See [Adding a New Predefined Application](https://learn.jamf.com/r/en-US/jamf-connect-documentation-current/Adding_a_New_SaaS_Application).

## Reaching private applications: custom DNS

An app on an internal network is not reachable just because an access policy names it. Its host names have to resolve, and public DNS will not resolve them — so Jamf Security Cloud needs a **custom DNS zone** naming your own authoritative name servers, and a gateway through which those name servers can be reached.

```hcl
resource "jamfplatform_security_cloud_dns_zone" "internal" {
  name = "Corp internal"

  # A parent domain and its subdomains are separate entries: "example.corp"
  # covers only the parent, "*.example.corp" only the subdomains.
  domains = ["example.corp", "*.example.corp"]

  # Each name server is paired with the gateway it is reachable through.
  # local.nearest_data_center comes from the locals block in the end-to-end
  # example above, and is the shared egress — a name server on your own network
  # is not reachable through it and needs a dedicated gateway instead.
  authoritative_name_servers = [
    { ip_address = "203.0.113.53", gateway_id = local.nearest_data_center },
    { ip_address = "203.0.113.54", gateway_id = local.nearest_data_center },
  ]
}
```

Queries matching the zone's domains go to one of its name servers by pseudo-random load balancing. Four rules here are Jamf's, not the provider's, and each one is a plan that applies and then fails to work:

- A **domain belongs to exactly one zone**, tenant-wide, and a name server IP address may appear only once per zone.
- The name server has to be reachable **through the gateway you name**, and **its address must not be in a reserved range** — Jamf Security Cloud refuses a private or loopback address with `Name server IP address not allowed`. Not every unrouted address is refused: a documentation range such as `203.0.113.0/24` is accepted, which is why the example above uses one. Jamf's own documentation attaches the restriction to the shared "Nearest Data Center" egress, but a dedicated *internet* gateway refuses a private address just the same, so do not plan on one lifting the rule. A name server on an internal address needs a gateway that actually reaches your network, which means the IPsec form below.
- Reverse lookups work by adding the `.in-addr.arpa` and `.ipv6.arpa` domains to the zone.
- **Changes to your own infrastructure need the zone changed too** — a new or renumbered name server, or an application on a domain the zone does not match.

Misconfiguring a zone cuts users off from the applications behind it, so the resource page says the same thing in stronger words. Jamf's [Configuring Custom DNS Zones](https://learn.jamf.com/r/en-US/jamf-security-cloud-setup-guide/Configuring_Custom_DNS_Zones) and [Troubleshooting Custom DNS Zones](https://learn.jamf.com/r/en-US/jamf-security-cloud-setup-guide/Troubleshooting_Custom_DNS_Zones) are the pages to read before and after.

### Gateways

Three kinds of gateway can be named as a zone's `gateway_id` or an app's routing target, and only two of them are yours to create:

- **Shared gateways** are Jamf-operated and available to every entitled tenant — "Nearest Data Center", which routes through whichever data centre is closest to the device, and a set of regional shared IP pools. Read them with the `..._ztna_shared_gateways` data source; they cannot be modified, deleted, or made a member of a group.
- A **dedicated gateway** is your own egress point, and a paid add-on. Setting the `ipsec` block builds a tunnel to your own VPN concentrator; omitting it provisions a pair of private egress IP addresses, reported back in `dedicated_egress_ip_addresses`.
- A **grouped gateway** is a failover and routing group over two or more dedicated gateways of the *same* form — all IPsec or all internet. Shared gateways are refused as members.

```hcl
resource "jamfplatform_security_cloud_ztna_gateway" "frankfurt" {
  name          = "Frankfurt DC"
  egress_region = "Europe - Germany"
  tenant_ids    = [var.security_cloud_tenant_id]

  contact = {
    name  = "Network Operations"
    email = "netops@example.com"
  }

  # Omitted here, so this is a dedicated internet gateway. Adding or removing the
  # block replaces the gateway — see "Immutable forms" below. The full IPsec form,
  # with its jamf_side and customer_side blocks, is on the
  # jamfplatform_security_cloud_ztna_gateway resource page.
  # ipsec = { ... }
}
```

`tenant_ids` is mandatory and every tenant named in it must belong to the same organization as the provider's credentials. Because no API lists an environment's tenants, an environment-scoped configuration has no way to fill it in from data — supply the tenant ID as an input, or configure the provider with `tenant_id`.

A gateway reports `PENDING` the moment it is created and carries no `dedicated_egress_ip_addresses` until Jamf has finished provisioning it, so an apply completing is not the same as a gateway being usable. Changing `egress_region` later re-provisions it: connectivity drops, the status returns to `PENDING`, and the egress addresses stay stale until it settles.

### Short names and fixed addresses

Two tenant-wide settings sit alongside the zones, both under **Custom DNS** in the portal, and both are single-instance resources:

```hcl
# Completes an incomplete host name for apps that only accept short names: a user
# asking for "wiki" is directed to "wiki.example.corp". Setting this is also what
# lets an access policy's hostnames be written as short names.
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

      # Traffic vectoring. Both are required on every mapping — note that the
      # admin UI's add dialog pre-selects Connect to ZTNA, so a mapping added
      # there and one written here do not start from the same value.
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

Between 1 and 500 mappings, each host name once, and at least one of `ipv4_addresses` or `ipv6_addresses` per entry. An empty `mappings` collection is not accepted: to remove them all, destroy the resource.

## Traffic matching is tenant-unique

A host name or address range belongs to **one** application across the whole tenant. Declaring one that another application already claims is rejected at apply, which makes a rename-by-replacement land badly: Terraform creates the new application before destroying the old one, and the create is refused because the old one still holds the names. Split such a change into two applies, or use `terraform state` to move the resource rather than replacing it.

Within one tenant Jamf resolves overlaps by specificity — the most granularly defined host name wins, so `images.app.example.com` matches an application declaring it ahead of one declaring `*.example.com`, and a custom application's host name takes precedence over the same name inside a predefined definition. That is what makes the pattern Jamf recommends work: start with one wildcard application to get traffic flowing, then carve specific applications out of it without touching the wildcard. See the [Network Engineer's Guide to Jamf Connect ZTNA](https://trusted.jamf.com/docs/network-engineers-guide-to-private-access).

Two further limits on `direct_ips_and_subnets`, both Jamf's: it needs the Jamf Trust app on the device, and direct-IP traffic does not appear in the access reports. Prefer host names where the application supports them.

## Delete ordering

Jamf Security Cloud enforces references on delete in three different ways, and knowing which is which saves a confusing destroy:

| Destroying | What happens |
|---|---|
| A **gateway** a DNS zone, grouped gateway or app still references | Refused, naming what holds it. |
| A **grouped gateway's member** while it is still in the group | Refused. |
| A **device group** an app assignment or UEM mapping still names | **Succeeds silently**, and quietly drops the group from every assignment that named it — which can leave an application assigned to nobody. |

The first has a Terraform-specific trap on top of the API's refusal. Dropping the reference *and* the gateway in a single apply does not work either, because Terraform sequences the destroy before the update that would have released the gateway — so the destroy still hits a live reference. Release it in one apply, destroy the gateway in the next.

The third is the dangerous one, because nothing fails. Check what references a device group before removing it. Note also that the built-in **Default Group** cannot be managed here at all: Jamf gives it no identifier and reserves the name. It shows up in the `..._device_groups` data source with `built_in = true` and a null `id`.

## Tenant-wide singletons

Three resources are one-per-tenant, and each behaves slightly differently when you destroy it:

- `dns_search_domain` — destroying it clears the search domain for the tenant.
- `dns_hostname_mappings` — owns the **whole set**; destroying it removes every mapping, and any mapping added outside Terraform is removed on the next apply.
- `uem_connect` — creating a second is refused, so **import** where one already exists rather than declaring a new one.

The integration's ID is not a value anyone would have written down, so read it off the data source — `data "jamfplatform_security_cloud_uem_connect" "existing" {}` reports it as `id` — or let `terraform query -generate-config-out=uem_connect.tf` write the import block for you from the list resource:

```shell
terraform import jamfplatform_security_cloud_uem_connect.jamf_pro 6a91b958619ef153a5a63d72
```

Declare each of them once. A second instance of any of them in one configuration will fight itself on every apply.

## Immutable forms

Three constructs pick a shape at create time that Jamf will not convert afterwards, so Terraform replaces the object instead:

- A **gateway** is IPsec or internet, decided by whether the `ipsec` block is present.
- A **ZTNA app** is predefined or custom, decided by whether `predefined_app_id` is set.
- A **UEM Connect** integration's vendor, address and authentication method are fixed; changing any of them replaces the integration, which briefly interrupts syncing.

## UEM Connect

UEM Connect syncs device inventory and group membership from Jamf Pro into Jamf Security Cloud, and signals device risk back. It is what makes the managed-device security requirement above mean anything, and what lets you keep deciding group membership in Jamf Pro.

```hcl
# The Jamf Pro tenant identifier is the one value here you cannot read off the
# UEM Connect screen, so read it rather than copying it between consoles.
data "jamfplatform_pro_tenant_id" "jamf_pro" {}

resource "jamfplatform_security_cloud_uem_connect" "jamf_pro" {
  uem_vendor = "JAMF_PRO"
  enabled    = true

  # Prefer platform_tenant: Jamf Security Cloud creates and manages its own
  # credentials on the named Jamf Pro tenant, so no secret is configured here.
  # The alternative, oauth, takes the client ID and secret of an API integration
  # you created on the Jamf Pro instance yourself, plus uem_server_url.
  platform_tenant = {
    tenant_id = data.jamfplatform_pro_tenant_id.jamf_pro.tenant_id
  }

  group_membership_mapping = {
    enabled = true

    # Order matters. Jamf evaluates the list in order and a device joins the
    # group of the first entry it matches, so put the most specific first. A
    # device matching nothing joins the default group.
    mappings = [
      {
        # A Jamf Pro group is written as computer_ or mobile_ followed by the
        # group's number. This composes out of a jamfplatform_device_group:
        #   "${jamfplatform_device_group.x.device_type}_${jamfplatform_device_group.x.jamf_pro_id}"
        uem_group_id            = "computer_12"
        security_cloud_group_id = jamfplatform_security_cloud_device_group.engineering.id
      },
    ]
  }
}

# Starts a sync immediately instead of waiting for the scheduled one. Jamf runs it
# in the background, so this returns as soon as the run has started and reports
# nothing about what it did — read latest_sync from the data source afterwards.
action "jamfplatform_security_cloud_uem_connect_synchronize" "now" {
  config {
    uem_connect_id = jamfplatform_security_cloud_uem_connect.jamf_pro.id
  }
}
```

`jamfplatform_pro_tenant_id` reaches the Jamf Pro namespace, so it works under a platform environment — where it resolves the Jamf Pro tenant in that environment — and under tenant scope pointed at the Jamf Pro tenant. It does **not** work under tenant scope pointed at the Security Cloud tenant, because Jamf Pro does not answer under that scope; supply the identifier as an input there. Note this is a different identifier from a gateway's `tenant_ids` above, which names Security Cloud tenants and has no data source to read it from.

Jamf Security Cloud does not check that either side of a mapping exists, so a wrong group number is accepted and simply never matches. The group configuration is replaced wholesale on every apply, so there is no way to leave it unmanaged: declaring `group_membership_mapping` replaces what it does not mention — an omitted or empty `mappings` clears every mapping — and **omitting the block entirely resets the whole group configuration to its defaults**. Manage it, or expect it cleared.

Importing an existing integration has one wrinkle worth knowing in advance: `user_data_field_mapping` and `group_membership_mapping` are captured from the tenant even though your configuration may not declare them, so run `terraform plan` straight after the import and write in what the plan shows you rather than letting the next apply clear them.

## Getting the configuration onto devices

Nothing above reaches a device on its own. Devices enrol with Jamf Security Cloud through an **activation profile** distributed alongside the Jamf Trust app, and the provider covers exactly one step of that: the portal's **Deploy to Jamf Pro** button, which creates the activation profile's configuration profile in Jamf Pro and scopes it.

```hcl
action "jamfplatform_security_cloud_activation_profile_deploy" "macos" {
  config {
    activation_profile_code = var.activation_profile_code
    os                      = "macos"
    jamf_pro_group_ids      = ["1", "2"]
  }
}
```

Three things about it. The **activation profile code** is issued by Jamf Security Cloud during activation profile setup and cannot be created or looked up from Terraform — it is the last path segment of the activation profile's deployment page — so it is an input. There is **one deployment per operating system** (`macos`, `ios_byod`, `ios_supervised`, `ios_unsupervised`), so cover more than one by invoking the action once for each, and name computer groups for `macos` and mobile device groups otherwise. And **scope only ever accumulates** — `jamf_pro_group_ids` adds groups and never removes one, and omitting it on a *first* deployment scopes the configuration profile to nothing while still reporting success. Narrow or clear a scope in Jamf Pro.

## Requirements

Jamf Security Cloud is reached under **either** scope: set `environment_id` (preferred) or `tenant_id` on the provider. Unlike the Platform Services constructs there is no version gate — Jamf Security Cloud is continuously deployed and a tenant can hold it without holding Jamf Pro.

Two things about authorisation are specific to this namespace:

- **Entitlement is not authentication.** A valid API integration can still be refused with `403 NOT_ENTITLED` when the tenant does not hold the product behind the construct — dedicated gateways, for instance, are a paid add-on. Every resource and action translates that into a named diagnostic; the three read-only catalogues — `..._content_categories`, `..._ztna_predefined_apps` and `..._ztna_shared_gateways` — surface the raw error instead.
- **Permissions are per construct**, granted in Jamf Account's permission picker. Each resource, data source and action page carries its own table naming the category, the row and the boxes to tick; the capabilities across this namespace are `ztna`, `device-groups`, `content-categories`, `custom-hostname-mappings`, `search-domains` and `uem-connect`.

## Further reading

| Topic | Page |
|---|---|
| The portal, end to end, in Jamf's words | [Jamf Security Cloud Portal Setup Guide](https://learn.jamf.com/r/en-US/jamf-security-cloud-setup-guide/RADAR_Portal) |
| What an access policy is made of | [Access Policy](https://learn.jamf.com/r/en-US/jamf-connect-documentation-current/Access_Policies) |
| Predefined and custom applications | [Adding a New Predefined Application](https://learn.jamf.com/r/en-US/jamf-connect-documentation-current/Adding_a_New_SaaS_Application), [Adding a New Custom Application](https://learn.jamf.com/r/en-US/jamf-connect-documentation-current/Adding_a_New_Enterprise_Application) |
| Gateway types, and what a shared gateway is | [Shared Internet Gateways](https://learn.jamf.com/r/en-US/jamf-connect-documentation-current/Internet_Cloud_Gateways), [Grouped Gateways](https://learn.jamf.com/r/en-US/jamf-connect-documentation-current/Grouped_Gateways) |
| Grouping gateways, and the routing strategies | [Creating a Group of Gateways](https://learn.jamf.com/r/en-US/jamf-connect-documentation-current/Creating_a_Grouped_Gateway) |
| Custom DNS zones | [DNS Zones](https://learn.jamf.com/r/en-US/jamf-security-cloud-setup-guide/Custom_DNS_Zones), [Configuring Custom DNS Zones](https://learn.jamf.com/r/en-US/jamf-security-cloud-setup-guide/Configuring_Custom_DNS_Zones), [Troubleshooting Custom DNS Zones](https://learn.jamf.com/r/en-US/jamf-security-cloud-setup-guide/Troubleshooting_Custom_DNS_Zones) |
| UEM Connect, and its settings | [UEM Connect](https://learn.jamf.com/r/en-US/jamf-security-cloud-setup-guide/UEM_Connect_Overview), [UEM Connect Settings Reference](https://learn.jamf.com/r/en-US/jamf-security-cloud-setup-guide/UEM_Connect_Settings), [UEM Connect Integration by Vendor](https://learn.jamf.com/r/en-US/jamf-security-cloud-setup-guide/UEM_Vendor_Integration) |
| Activation profiles | [Activation Profiles](https://learn.jamf.com/r/en-US/jamf-security-cloud-setup-guide/Activation_Profiles), [Creating an Activation Profile](https://learn.jamf.com/r/en-US/jamf-security-cloud-setup-guide/Creating_Activation_Profiles) |
| The policies this provider does not manage yet | [Policies](https://learn.jamf.com/r/en-US/jamf-security-cloud-setup-guide/Policies) |
| How the routing fabric actually works | [Network Engineer's Guide to Jamf Connect ZTNA](https://trusted.jamf.com/docs/network-engineers-guide-to-private-access) |
