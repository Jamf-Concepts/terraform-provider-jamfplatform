# terraform-provider-jamf-platform

Provides resources and data sources for managing [Jamf Platform Services](https://developer.jamf.com/platform-api/):

* [Compliance Benchmark Engine](https://learn.jamf.com/en-US/bundle/jamf-compliance-benchmarks-configuration-guide/page/Compliance_Benchmarks_Configuration_Guide.html)
  * [API Reference](https://developer.jamf.com/platform-api/reference/getbenchmark)
* Unified Inventory
  * [API Reference](https://developer.jamf.com/platform-api/reference/computer-inventory)
* [Blueprints](https://learn.jamf.com/en-US/bundle/jamf-pro-blueprints-configuration-guide/page/Jamf_Pro_Blueprints_Configuration_Guide.html)
  * [API Reference](https://developer.jamf.com/platform-api/reference/blueprints-1)

Note that some of these APIs are only available in private beta. Provider stability, functionality and schemas are subject to change without notice.

**This repository also includes a Go client for direct API access and scripting.**

## Requirements

* Terraform 0.12 or later, or OpenTofu 1.6.0 or later

## Using the Provider in your own Terraform Projects

The jamfplatform provider is published in the [Hashicorp](https://registry.terraform.io/providers/Jamf-Concepts/jamfplatform) and [OpenTofu](https://search.opentofu.org/provider/jamf-concepts/jamfplatform) registries.

For usage instructions and provider block/variable reference, refer to the registry link above for your platform of choice.

---

## Provider Configuration Reference and Example Usage

Refer to the [documentation](https://registry.terraform.io/providers/Jamf-Concepts/jamfplatform/latest/docs) for a full list of resources and data sources, their usage and Terraform examples.

---

## Using the Go Client in Your Own Go Projects

The provider includes a comprehensive Go client for interacting with the Jamf Platform API. You can import and use the client directly in your own Go projects for scripting or automation against the services supported by the Jamf Platform API.

For example, to get a list of current Compliance Baselines the Compliance Benchmark Engine currently supports from the mSCP:

```go
import "github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"

func main() {
    apiClient := client.NewClient("https://region.apigw.jamf.com", "your-client-id", "your-client-secret")
    // Use apiClient to call API methods, e.g.:
    baselines, err := apiClient.GetCBEngineBaselines(context.Background())
    // ...
}
```

See the [examples/client/](./examples/client/) directory for full working Go examples.

---

## Feedback & Discussion

Please contact the project principles via [GitHub Issues](https://github.com/Jamf-Concepts/terraform-provider-jamfplatform/issues).

The Jamf Terraform community has discussions in #terraform-provider-jamfpro on [MacAdmins Slack](https://www.macadmins.org/). This channel is primarily focused on discussion and community support relating to the [jamfpro](https://github.com/deploymenttheory/terraform-provider-jamfpro) provider that is owned and maintained by our friends, [Deployment Theory](https://github.com/deploymenttheory).

## Included components

The following third party acknowledgements and licenses are incorporated by reference:

* [Terraform Plugin Framework](https://github.com/hashicorp/terraform-plugin-framework) ([MPL](https://github.com/hashicorp/terraform-plugin-framework?tab=MPL-2.0-1-ov-file))
* [Terraform Plugin Log](https://github.com/hashicorp/terraform-plugin-log) ([MPL](https://github.com/hashicorp/terraform-plugin-log?tab=MPL-2.0-1-ov-file))

&nbsp;

*Copyright 2025, Jamf Software LLC.*
