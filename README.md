# terraform-provider-jamf-platform

Provides resources and data sources for managing [Jamf Platform Services](https://developer.jamf.com/platform-api/docs/getting-started-with-the-platform-api):

* [Compliance Benchmark Engine](https://learn.jamf.com/en-US/bundle/jamf-compliance-benchmarks-configuration-guide/page/Compliance_Benchmarks_Configuration_Guide.html)
* Unified Inventory

**This repository also includes a Go client for direct API access and scripting.**

## Requirements

* Terraform 1.3 or later

## Using the Provider in your own Terraform Projects

### If using the Terraform Registry

```hcl
terraform {
  required_providers {
    jamfplatform = {
      source  = "Jamf-Concepts/jamfplatform"
      version = ">= 1.0.0"
    }
  }
}

provider "jamfplatform" {
  region = "us" # or "eu", "apac"

  cbengine = {
    client_id     = "example-cbengine-client-id"
    client_secret = "example-cbengine-client-secret"
  }

  inventory = {
    client_id     = "example-inventory-client-id"
    client_secret = "example-inventory-client-secret"
  }
}
```

### If using a local install (see below)

```hcl
terraform {
  required_providers {
    jamfplatform = {
      source  = "local/Jamf-Concepts/jamfplatform"
      version = ">= 1.0.0"
    }
  }
}

provider "jamfplatform" {
  region = "us" # or "eu", "apac"

  cbengine = {
    client_id     = "example-cbengine-client-id"
    client_secret = "example-cbengine-client-secret"
  }

    inventory = {
    client_id     = "example-inventory-client-id"
    client_secret = "example-inventory-client-secret"
  }

}
```

---

### Initialize Terraform

```bash
terraform init
```

Terraform will detect the provider and you're good to go!

**Note: If you are updating this provider manually, you should also delete the lock file:**

```bash
rm .terraform.lock.hcl
terraform init
```

## Manual Installation

### Step 1: Download the Release Zip

Pick your platform and architecture from the [latest releases](https://github.com/Jamf-Concepts/terraform-provider-jamfplatform/releases/latest) page:

* If running on an Apple Silicon Mac, download `...darwin_arm64.zip`
* If running on an Intel Mac, download `...darwin_amd64.zip`

---

### Step 2: Extract to your local Terraform Plugin Directory

The plugin must be extracted to the correct location for Terraform to find and use it. You must also remove the Quarantine attribute on macOS.

#### Example (macOS arm64)

```bash
cd ~/Downloads
mkdir -p ~/.terraform.d/plugins/local/Jamf-Concepts/jamfplatform/1.1.2/darwin_arm64
unzip terraform-provider-jamfplatform_1.1.2_darwin_arm64.zip -d ~/.terraform.d/plugins/local/Jamf-Concepts/jamfplatform/1.1.2/darwin_arm64
xattr -r -d com.apple.quarantine ~/.terraform.d/plugins
```

This will result in:

```bash
~/.terraform.d/plugins/
└── local/
    └── Jamf-Concepts/
        └── jamfplatform/
            └── 1.1.2/
                └── darwin_arm64/
                    └── terraform-provider-jamfplatform_v1.1.2
```

### Step 3: Set up a local file system mirror

Terraform needs to know it will be using a locally installed plugin. Create the file: `~/.terraform.d/terraform.tfrc`

* `nano ~/.terraform.d/terraform.tfrc`

It must contain the following contents:

```hcl
provider_installation {
  filesystem_mirror {
    path    = ~"/.terraform.d/plugins"
    include = ["local/Jamf-Concepts/*"]
  }
  direct {
    exclude = ["local/Jamf-Concepts/*"]
  }
}
```

* In nano, use **ctrl+x** then enter **y** and **return** to save.

That's it! You're ready to use the provider!

---

## Provider Configuration Reference and Example Usage

Refer to the [documentation](./docs) and the [examples](./examples/) directories for full usage and Terraform examples.

---

## Using the Go Client in Your Own Go Projects

You can import and use the Go client directly in your own Go code for scripting or automation against the services supported by the Jamf Platform API.

For example, go get a list of current Compliance Baselines from the mSCP:

```go
import "github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"

func main() {
    apiClient := client.NewCBEngineClient("us", "your-client-id", "your-client-secret")
    // Use apiClient to call API methods, e.g.:
    baselines, err := apiClient.GetCBEngineBaselines(context.Background())
    // ...
}
```

See the [examples/client/](./examples/client/) directory for full working Go examples.

---
