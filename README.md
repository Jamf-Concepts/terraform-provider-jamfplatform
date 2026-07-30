# terraform-provider-jamfplatform

> **📘 Start here — [Managing the Jamf Platform with Terraform: the Jamf Platform provider](https://concepts.jamf.com/en/guides/infrastructure-as-code/managing-the-jamf-platform-with-terraform-the-jamf-platform-provider/)**
>
> The official getting-started guide on Jamf Concepts. Covers installing Terraform, creating API credentials, configuring the provider, writing your first device groups / compliance benchmarks / blueprints, applying a configuration, and bringing an existing tenant under management. New to this provider or to Terraform? Read that first.

Provides resources and data sources for managing the products and services available through the [Jamf Platform API](https://developer.jamf.com/platform-api/):

* [Compliance Benchmark Engine](https://learn.jamf.com/en-US/bundle/jamf-compliance-benchmarks-configuration-guide/page/Compliance_Benchmarks_Configuration_Guide.html)
  * [API Reference](https://developer.jamf.com/platform-api/reference/getbenchmark)
* [Blueprints](https://learn.jamf.com/en-US/bundle/jamf-pro-blueprints-configuration-guide/page/Jamf_Pro_Blueprints_Configuration_Guide.html)
  * [API Reference](https://developer.jamf.com/platform-api/reference/listblueprints)
* Device Groups
  * [API Reference](https://developer.jamf.com/platform-api/reference/get_v1-tenant-tenantid-device-groups)
* [Devices](https://developer.jamf.com/platform-api/reference/get_v1-tenant-tenantid-devices)
* [Device Actions](https://developer.jamf.com/platform-api/reference/post_v1-tenant-tenantid-devices-id-check-in)

It additionally provides resources and data sources for [Jamf Pro](https://developer.jamf.com/platform-api/reference/get_v1-tenant-tenantid-account-groups) under the `jamfplatform_pro_*` namespace, with further Jamf products to follow. See the **Supported Jamf products** section below for the per-product tenant version targets.

Note that the Platform API is currently in public beta. Provider stability, functionality and schemas are subject to change without notice.

## Acknowledgements

This provider builds on a path charted by [Deployment Theory](https://github.com/deploymenttheory) and their [`terraform-provider-jamfpro`](https://github.com/deploymenttheory/terraform-provider-jamfpro) — first released in early 2024, it has grown into the most comprehensive community Terraform provider for Jamf and the one the community rallies behind. It proved the depth of demand for managing Jamf as code and sets the bar for what a Jamf provider could be; this provider would not exist in its current form without that groundwork. `terraform-provider-jamfpro` remains an independent, actively maintained project, and we're grateful to its maintainers for the example they produce.

## Requirements

* Terraform >= 1.13.0, or OpenTofu >= 1.6.0

### Supported Jamf products and tenant version targets

The provider groups resources by the Jamf product they target. Some products are versioned at the customer-tenant level (the API spec the provider was generated from may be newer than what a given tenant is running); others are continuously-deployed Jamf Platform microservices with no tenant version concept.

| Product | Resource namespace | Built against API as of | Notes |
|---------|--------------------|--------------------------|-------|
| Jamf Pro | `jamfplatform_pro_*` | **11.29.0** (see [`ProviderMinJamfProVersion`](./internal/providerdata/providerdata.go) for the current source-tree value) | Tenants below this version emit an advisory warning at apply time. Individual resources that depend on newer endpoints declare their own `minJamfProVersion` and hard-fail Configure on unsupported tenants. |
| Jamf Platform Services (Blueprints, Device Groups, Devices, Device Actions, Compliance Benchmarks) | resources without a product-name prefix (e.g. `jamfplatform_blueprints_blueprint`, `jamfplatform_device_group`) | continuously-deployed | No tenant version requirement. No version fetch is performed against tenants that use only these resources. |

Further Jamf products are expected to be added; each will get its own row, namespace, and version constant.

**Credentials are the same across all products**: a single OAuth client (`JAMFPLATFORM_CLIENT_ID` / `JAMFPLATFORM_CLIENT_SECRET`) scoped to the relevant API areas on your tenant.

## Using the Provider in your own Terraform Projects

The jamfplatform provider is published in the [Hashicorp](https://registry.terraform.io/providers/Jamf-Concepts/jamfplatform) and [OpenTofu](https://search.opentofu.org/provider/jamf-concepts/jamfplatform) registries.

For a step-by-step walkthrough — from installing Terraform through to applying your first configuration and importing an existing tenant — see the Jamf Concepts guide: [**Managing the Jamf Platform with Terraform: the Jamf Platform provider**](https://concepts.jamf.com/en/guides/infrastructure-as-code/managing-the-jamf-platform-with-terraform-the-jamf-platform-provider/).

For provider block/variable reference, refer to the registry link above for your platform of choice.

---

## Provider Configuration Reference and Example Usage

Refer to the [documentation](https://registry.terraform.io/providers/Jamf-Concepts/jamfplatform/latest/docs) for a full list of resources and data sources, their usage and Terraform examples.

---

## Contributing

Contributions are welcome. See [CONTRIBUTING.md](./CONTRIBUTING.md) and [TESTING.md](./TESTING.md) for the full workflow. In short, for changes that add or modify resources, data sources, list resources, actions, or functions:

1. **Add Go unit tests** — schema validation, input builders, state builders, and (where relevant) state upgraders.
2. **Add Go acceptance tests** — `resource_acceptance_test.go` (or `datasource_acceptance_test.go`) with the `//go:build acceptance` tag, using factories from `internal/testhelpers`. Run locally with `make testacc` against a test tenant.
3. **Update examples** — add `.tf` files under the appropriate `examples/` subdirectory.
4. **Run `make generate`** — regenerates `docs/` from schema descriptions and applies copyright headers.
5. **CI** — `.github/workflows/integration-tests.yml` runs build, lint, docs-generation check, and the Go unit suite on every PR. The Go acceptance suite runs against a real tenant after a reviewer approves the `acceptance` environment gate.

For bug reports, feature requests, or general discussion, please use [GitHub Issues](https://github.com/Jamf-Concepts/terraform-provider-jamfplatform/issues).

---

## Feedback & Discussion

Please contact the project principles via [GitHub Issues](https://github.com/Jamf-Concepts/terraform-provider-jamfplatform/issues).

The Jamf Terraform community has discussions in #terraform-provider-jamfpro on [MacAdmins Slack](https://www.macadmins.org/). This channel is primarily focused on discussion and community support relating to the [jamfpro](https://github.com/deploymenttheory/terraform-provider-jamfpro) provider that is owned and maintained by our friends, [Deployment Theory](https://github.com/deploymenttheory).

## Included components

The following third party acknowledgements and licenses are incorporated by reference:

* [Jamf Platform Go SDK](https://github.com/Jamf-Concepts/jamfplatform-go-sdk) ([MIT](https://github.com/Jamf-Concepts/jamfplatform-go-sdk?tab=MIT-1-ov-file))
* [Terraform Plugin Framework](https://github.com/hashicorp/terraform-plugin-framework) ([MPL](https://github.com/hashicorp/terraform-plugin-framework?tab=MPL-2.0-1-ov-file))
* [Terraform Plugin Log](https://github.com/hashicorp/terraform-plugin-log) ([MPL](https://github.com/hashicorp/terraform-plugin-log?tab=MPL-2.0-1-ov-file))

&nbsp;

*Copyright 2026, Jamf Software LLC.*
