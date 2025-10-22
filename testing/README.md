# Integration Testing

This directory contains integration tests for the `terraform-provider-jamfplatform` provider. The tests validate resources and data sources against real Jamf Platform APIs using Terraform's native testing framework.

## Table of Contents

- [Overview](#overview)
- [Test Structure](#test-structure)
- [Running Tests Locally](#running-tests-locally)
- [CI/CD Integration](#cicd-integration)
- [Adding New Tests](#adding-new-tests)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

## Overview

The integration test suite uses [Terraform's native test framework](https://developer.hashicorp.com/terraform/language/tests) to validate provider functionality. Tests are organized into two main categories:

- **Data Sources**: Read-only operations that fetch data from Jamf Platform
- **Resources**: Create, update, and delete operations for managing Jamf Platform configurations

### Test Execution Flow

1. **Setup**: Terraform compiles the provider binary and initializes the testing workspace
2. **Plan Tests**: Data source tests run with `terraform plan` to validate read operations
3. **Apply Tests**: Resource tests run with `terraform apply` to create/update/delete resources
4. **Cleanup**: Resources are automatically destroyed after test completion

## Test Structure

```plaintext
testing/
├── integration.tftest.hcl    # Main test orchestration file
├── provider.tf                # Provider configuration (shared via symlinks)
├── variables.tf               # Input variables for authentication and test ID
├── data_sources/              # Data source tests (read-only)
│   ├── provider.tf            # → Symlink to ../provider.tf
│   ├── variables.tf           # → Symlink to ../variables.tf
│   ├── blueprints_components.tf
│   ├── cbengine_baselines.tf
│   ├── inventory_computers.tf
│   └── ...
└── resources/                 # Resource tests (CRUD operations)
    ├── provider.tf            # → Symlink to ../provider.tf
    ├── variables.tf           # → Symlink to ../variables.tf
    ├── main.tf                # Shared resources (Jamf Pro groups)
    ├── blueprints_blueprint_passcode_policy.tf
    ├── cbengine_benchmark_all.tf
    └── ...
```

### Key Files

| File | Purpose |
|------|---------|
| `integration.tftest.hcl` | Orchestrates test execution with `run` blocks |
| `provider.tf` | Configures multiple provider instances (default, inventory alias, jamfpro) |
| `variables.tf` | Defines authentication credentials and test ID variable |
| `data_sources/*.tf` | Individual data source test configurations |
| `resources/*.tf` | Individual resource test configurations |
| `resources/main.tf` | Shared test infrastructure (device groups) |

## Running Tests Locally

### Prerequisites

1. **Go**: Version specified in `go.mod`
2. **Terraform**: Latest stable version
3. **Authentication credentials**:
   - Jamf Platform OAuth client (compliance/blueprints APIs)
   - Jamf Platform OAuth client (inventory APIs)
   - Jamf Pro OAuth client

### Step 1: Set Environment Variables

Export the required authentication variables:

```bash
export TF_VAR_jamfplatform_base_url="https://your-instance.jamfcloud.com"
export TF_VAR_jamfplatform_client_id="your-client-id"
export TF_VAR_jamfplatform_client_secret="your-client-secret"

export TF_VAR_jamfplatform_inventory_client_id="your-inventory-client-id"
export TF_VAR_jamfplatform_inventory_client_secret="your-inventory-client-secret"

export TF_VAR_jamfpro_instance_fqdn="your-instance.jamfcloud.com"
export TF_VAR_jamfpro_client_id="your-jamfpro-client-id"
export TF_VAR_jamfpro_client_secret="your-jamfpro-client-secret"

# Optional: Unique test ID to avoid naming conflicts
export TF_VAR_test_id="$(uuidgen)"
```

### Step 2: Build and Install Provider Locally

From the repository root:

```bash
# Build the provider
go build -buildvcs=false

# Install to local Terraform plugins directory
mkdir -p ~/.terraform.d/plugins/local/jamf/jamfplatform/0.1.0/darwin_arm64/
cp ./terraform-provider-jamfplatform ~/.terraform.d/plugins/local/jamf/jamfplatform/0.1.0/darwin_arm64/
chmod +x ~/.terraform.d/plugins/local/jamf/jamfplatform/0.1.0/darwin_arm64/terraform-provider-jamfplatform
```

**Note**: Replace `darwin_arm64` with your platform architecture:

- macOS Intel: `darwin_amd64`
- macOS Apple Silicon: `darwin_arm64`
- Linux: `linux_amd64`
- Windows: `windows_amd64`

### Step 3: Initialize Terraform

```bash
cd testing
terraform init -upgrade
```

### Step 4: Run Tests

```bash
# Run all tests
terraform test -verbose -parallelism=1

# Run with auto-approve (no prompts)
terraform test -verbose -parallelism=1
```

**Important**: Use `-parallelism=1` to avoid race conditions when creating/destroying resources.

## CI/CD Integration

Tests automatically run on pull requests via GitHub Actions (`.github/workflows/integration-tests.yml`).

### Workflow Triggers

The workflow runs when PRs modify:

- Go source files (`**.go`)
- Terraform files (`**.tf`)
- Test directory (`testing/**`)
- Go module files (`go.mod`, `go.sum`)
- Workflow file itself

### Workflow Steps

1. **Checkout Repository**: Fetches the PR code
2. **Setup Go**: Installs Go version from `go.mod`
3. **Setup Terraform**: Installs latest Terraform
4. **Compile Provider**: Builds and installs provider to local plugin directory
5. **Initialize Terraform**: Runs `terraform init` in testing directory
6. **Generate UUID**: Creates unique test ID to avoid resource naming conflicts
7. **Run Tests**: Executes `terraform test` with authentication from GitHub Secrets

### Required GitHub Secrets

Configure these secrets in your repository settings:

| Secret Name | Description |
|-------------|-------------|
| `JAMFPLATFORM_BASE_URL` | Jamf Platform instance URL |
| `JAMFPLATFORM_CLIENT_ID` | OAuth client ID for compliance/blueprints |
| `JAMFPLATFORM_CLIENT_SECRET` | OAuth client secret for compliance/blueprints |
| `JAMFPLATFORM_INVENTORY_CLIENT_ID` | OAuth client ID for inventory APIs |
| `JAMFPLATFORM_INVENTORY_CLIENT_SECRET` | OAuth client secret for inventory APIs |
| `JAMFPRO_INSTANCE_FQDN` | Jamf Pro instance FQDN |
| `JAMFPRO_CLIENT_ID` | Jamf Pro OAuth client ID |
| `JAMFPRO_CLIENT_SECRET` | Jamf Pro OAuth client secret |

## Adding New Tests

Follow these steps to add tests for a new resource or data source:

### Adding a Data Source Test

**Step 1: Create the test file** in `testing/data_sources/`:

```bash
cd testing/data_sources
touch your_data_source.tf
```

**Step 2: Write your test configuration**:

```hcl
# data_sources/your_data_source.tf
data "jamfplatform_your_data_source" "test_example" {
  # Add required arguments
  filter = "example"
}

# Optional: Add outputs to verify data
output "test_your_data_source_results" {
  value = data.jamfplatform_your_data_source.test_example
}
```

**Step 3: Verify symlinks exist**:

```bash
# In data_sources/ directory
ls -la provider.tf variables.tf
# Should show symlinks to ../provider.tf and ../variables.tf
```

If symlinks are missing, create them:

```bash
ln -s ../provider.tf provider.tf
ln -s ../variables.tf variables.tf
```

**Step 4: Test locally**:

```bash
cd ../
terraform test -verbose -parallelism=1
```

Data source tests run during the `test_data_sources` run block with `command = plan`.

### Adding a Resource Test

**Step 1: Create the test file** in `testing/resources/`:

```bash
cd testing/resources
touch your_resource.tf
```

**Step 2: Write your test configuration**:

```hcl
# resources/your_resource.tf
resource "jamfplatform_your_resource" "test_example" {
  name        = "Terraform Test Example ${var.test_id}"
  description = "Managed by Terraform integration tests"
  
  # Add required arguments
  setting = "example_value"
  
  # Use target groups from main.tf if needed
  device_groups = [
    data.jamfpro_group.test_target_computer_group.group_platform_id
  ]
}

# Optional: Add outputs to verify creation
output "test_your_resource_id" {
  value = jamfplatform_your_resource.test_example.id
}
```

**Step 3: Verify symlinks exist** (same as data sources):

```bash
ln -s ../provider.tf provider.tf
ln -s ../variables.tf variables.tf
```

**Step 4: Test locally**:

```bash
cd ../
terraform test -verbose -parallelism=1
```

Resource tests run during the `test_resources` run block with `command = apply`.

### Using Shared Test Infrastructure

The `resources/main.tf` file provides shared infrastructure that can be referenced in your tests:

```hcl
# Available shared resources:
data.jamfpro_group.test_target_computer_group.group_platform_id
data.jamfpro_group.test_target_mobile_device_group.group_platform_id
```

These groups are created via the `jamfpro` provider and can be used as target groups for blueprints, benchmarks, and other resources.

### Using Provider Aliases

Some resources require specific provider configurations:

```hcl
# Use the inventory provider alias
data "jamfplatform_inventory_computers" "test" {
  provider = jamfplatform.inventory
}
```

The available provider aliases are:

- `jamfplatform` (default) - For compliance/blueprints APIs
- `jamfplatform.inventory` - For inventory APIs
- `jamfpro` - For Jamf Pro Classic/UAPI

## Best Practices

### Naming Conventions

- **Include test_id**: All resources must include `${var.test_id}` in names to avoid conflicts:

  ```hcl
  name = "Terraform Test Passcode Policy ${var.test_id}"
  ```

- **Use descriptive prefixes**: Start names with "Terraform Test" for easy identification

### Resource Organization

- **One resource type per file**: Each `.tf` file should test one resource/data source
- **Use meaningful filenames**: Match the resource/data source type (e.g., `blueprints_blueprint_passcode_policy.tf`)
- **Group related tests**: Use `for_each` when testing variations of the same resource

### Test Design

- **Test realistic scenarios**: Use configuration values that represent real-world usage
- **Validate dependencies**: If your resource depends on data sources, include them in tests
- **Handle provider aliases**: Explicitly specify provider when using non-default configurations
- **Include required fields**: Ensure all required arguments are populated

### Cleanup

- **Automatic cleanup**: Terraform test framework destroys resources automatically after tests complete
- **Use unique IDs**: The `test_id` variable ensures no conflicts between parallel test runs
- **Verify destruction**: Check test output to confirm resources were successfully destroyed

## Troubleshooting

### Common Issues

#### Provider Not Found

```text
Error: Failed to query available provider packages
```

**Solution**: Rebuild and reinstall the provider binary (see [Step 2](#step-2-build-and-install-provider-locally))

#### Authentication Failures

```text
Error: Failed to authenticate with Jamf Platform
```

**Solution**: Verify environment variables are exported correctly:

```bash
env | grep TF_VAR_
```

#### Symlink Errors

```text
Error: Module not installed
```

**Solution**: Create missing symlinks:

```bash
cd data_sources/  # or resources/
ln -s ../provider.tf provider.tf
ln -s ../variables.tf variables.tf
```

#### Resource Naming Conflicts

```text
Error: Resource already exists
```

**Solution**: Ensure `test_id` variable is set to a unique value:

```bash
export TF_VAR_test_id="$(uuidgen)"
```

#### Parallelism Issues

```text
Error: Resource still in use
```

**Solution**: Always run tests with `-parallelism=1`:

```bash
terraform test -verbose -parallelism=1
```

### Debug Mode

Enable detailed Terraform logging:

```bash
export TF_LOG=DEBUG
export TF_LOG_PATH=./terraform-debug.log
terraform test -verbose -parallelism=1
```

### Selective Test Execution

To test only data sources or resources:

```bash
# Test only data sources (plan-only)
cd data_sources
terraform plan

# Test only resources (apply)
cd resources
terraform apply
```

## Additional Resources

- [Terraform Testing Documentation](https://developer.hashicorp.com/terraform/language/tests)
- [Jamf Platform Provider Documentation](https://registry.terraform.io/providers/jamf/jamfplatform/latest/docs)
- [Jamf Pro Provider Documentation](https://registry.terraform.io/providers/deploymenttheory/jamfpro/latest/docs)

## Contributing

When submitting a PR that adds new resources or data sources:

1. ✅ Add integration tests following this guide
1. ✅ Verify tests pass locally with `terraform test`
1. ✅ Ensure all symlinks are created correctly
1. ✅ Use the `test_id` variable in resource names
1. ✅ Document any new test dependencies or requirements

The GitHub Actions workflow will automatically run your tests on PR submission.
