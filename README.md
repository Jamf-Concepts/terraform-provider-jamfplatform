# Jamf Compliance Benchmark Engine Terraform Provider

Provides resources and data sources for managing [Jamf Compliance Benchmark](https://learn.jamf.com/en-US/bundle/jamf-compliance-benchmarks-configuration-guide/page/Compliance_Benchmarks_Configuration_Guide.html) baselines, rules, and benchmarks via the [Jamf Compliance Benchmark Engine API](https://developer.jamf.com/platform-api/reference).

**This repository also includes a Go client for direct API access and scripting.**

![image](https://github.com/user-attachments/assets/f315d765-4346-46ea-b66a-1b127a6db9c3)

* Create, Update and Delete Benchmark resources (updates force a destroy and create until PUT is supported by the API)
* Read Benchmark, Baesline and Rule data sources

## Requirements

* Terraform 1.3 or later

## Using the Provider in your own Terraform Projects

### If using the Terraform Registry

```hcl
terraform {
  required_providers {
    jamfcompliancebenchmarkengine = {
      source  = "Jamf-Concepts/jamfcompliancebenchmarkengine"
      version = ">= 1.0.0"
    }
  }
}

provider "jamfcompliancebenchmarkengine" {
  region        = "us" # or "eu", "apac"
  client_id     = "example-client-id"
  client_secret = "example-client-secret"
}
```

### If using a local install (see below)

```hcl
terraform {
  required_providers {
    jamfcompliancebenchmarkengine = {
      source  = "local/Jamf-Concepts/jamfcompliancebenchmarkengine"
      version = ">= 1.0.0"
    }
  }
}

provider "jamfcompliancebenchmarkengine" {
  region        = "us" # or "eu", "apac"
  client_id     = "example-client-id"
  client_secret = "example-client-secret"
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

Pick your platform and architecture from the [latest releases](https://github.com/Jamf-Concepts/terraform-provider-jamfcompliancebenchmarkengine/releases/latest) page:

* If running on an Apple Silicon Mac, download `...darwin_arm64.zip`
* If running on an Intel Mac, download `...darwin_amd64.zip`

---

### Step 2: Extract to your local Terraform Plugin Directory

The plugin must be extracted to the correct location for Terraform to find and use it. You must also remove the Quarantine attribute on macOS.

#### Example (macOS arm64)

```bash
cd ~/Downloads
mkdir -p ~/.terraform.d/plugins/local/Jamf-Concepts/jamfcompliancebenchmarkengine/1.0.0/darwin_arm64
unzip terraform-provider-jamfcompliancebenchmarkengine_1.0.0_darwin_arm64.zip -d ~/.terraform.d/plugins/local/Jamf-Concepts/jamfcompliancebenchmarkengine/1.0.0/darwin_arm64
xattr -r -d com.apple.quarantine ~/.terraform.d/plugins
```

This will result in:

```bash
~/.terraform.d/plugins/
└── local/
    └── Jamf-Concepts/
        └── jamfcompliancebenchmarkengine/
            └── 1.0.0/
                └── darwin_arm64/
                    └── terraform-provider-jamfcompliancebenchmarkengine_v1.0.0
```

### Step 3: Set up a local file system mirror

Terraform needs to know it will be using a locally installed plugin. Create the file: `~/.terraform.d/terraform.tfrc`

* `nano ~/.terraform.d/terraform.tfrc`

It must contain the following contents:

```hcl
provider_installation {
  filesystem_mirror {
    path    = "~/.terraform.d/plugins"
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

You can import and use the Go client directly in your own Go code for scripting or automation against the Jamf Compliance Benchmark Engine API.

```go
import "github.com/Jamf-Concepts/terraform-provider-jamfcompliancebenchmarkengine/internal/client"

func main() {
    apiClient := client.NewClient("us", "your-client-id", "your-client-secret")
    // Use apiClient to call API methods, e.g.:
    baselines, err := apiClient.GetBaselines(context.Background())
    // ...
}
```

See the [examples/client/](./examples/client/) directory for full working Go examples.

---
