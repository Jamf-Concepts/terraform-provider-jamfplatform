# Style Guide

Code style conventions for the Terraform Provider for Jamf Platform.

## Go Conventions

- Follow standard Go conventions and idiomatic patterns.
- Run `go fmt ./...` and `golangci-lint run` before committing.
- Use clear, descriptive names for variables, functions, and types.
- Every exported constant, function, variable set, and type must have a short comment describing its purpose.
- Do not add comments inside type definitions or function bodies.

## Dependencies

Only use native Go, `golang.org/x` packages, and Terraform Plugin Framework packages. Do not introduce third-party dependencies.

## Resource Package File Conventions

Resource packages live under `internal/resources/<domain>/<resource>/` and use resource-agnostic filenames:

| File | Purpose |
|------|---------|
| `resource.go` | Schema definition and boilerplate |
| `crud.go` | Create, Read, Update, Delete, and ImportState |
| `model_types.go` | Terraform model structs |
| `schema_types.go` | Attribute type maps for `ObjectValue`/`ListValue` state |
| `mappings.go` | Lookup tables and name mappings |
| `input_builders.go` | Build API request payloads from Terraform model data |
| `state_builders.go` | Map API responses to Terraform state |
| `helpers.go` | Resource-specific helper functions |
| `plan_modifiers.go` | Schema plan modifiers (if needed) |
| `validators.go` | Schema validators (if needed) |
| `list_resource.go` | List resource implementation |
| `data_source.go` | Data source implementation |

### Optional split-outs for complex resources

- `endpoints_builders.go` / `endpoints_state.go` — when endpoint logic dominates.
- `nested_builders.go` / `nested_state.go` — for large nested payloads.

### Data-source-only packages

Packages that only contain a data source use `model_types.go` for their model structs and `data_source.go` for the implementation.

## Test File Conventions

| File | Purpose |
|------|---------|
| `schema_test.go` | Schema validation (every package) |
| `helpers_test.go` | Helper function tests |
| `input_builders_test.go` | Input builder tests |
| `state_builders_test.go` | State builder tests |
| `resource_acceptance_test.go` | Acceptance tests (`//go:build acceptance`) |
| `datasource_acceptance_test.go` | Data source acceptance tests |

## Client Conventions

### Versioned naming

Client types and functions use explicit version suffixes to support multiple API versions:

```go
// V1 functions
func (c *Client) CreateDeviceGroupV1(ctx context.Context, req *DeviceGroupCreateRepresentationV1) (*DeviceGroupCreateResponseV1, error)

// V2 functions added alongside V1
func (c *Client) CreateCBEngineBenchmarkV2(ctx context.Context, req *CBEngineBenchmarkRequestV2) (*CBEngineBenchmarkResponseV2, error)
```

When endpoints are upgraded, add new versioned types/functions alongside existing ones. Resources migrate at their own pace.

### Type naming

Request and response types include the version suffix and follow the pattern `<Domain><Entity><Purpose><Version>`:

```go
type DeviceGroupCreateRepresentationV1 struct { ... }
type CBEngineBenchmarkResponseV2 struct { ... }
```

## Schema Guidelines

- Keep schemas inline and as flat as possible.
- Favor nested attributes (`SingleNestedAttribute`, `SetNestedAttribute`, `ListNestedAttribute`) over blocks.

### Sets vs Lists

- **Sets** for user-supplied unordered collections where deduplication and order-independent comparison matter (e.g. `members`, `criteria`, `raw_component`).
- **Lists** for computed API results that are read-only. Sets require element hashing which adds overhead with no benefit when the user doesn't control the values.

Data source attributes returning API data should always use lists.

## Error Handling

- Use `helpers.IsNotFoundError(err)` for 404 detection in Read/Delete operations.
- Use `helpers.PollUntil(ctx, interval, checker)` for async operations that need polling.
- Wrap errors with `fmt.Errorf("context: %w", err)` to preserve the error chain.

## Naming Patterns

### Resources

Terraform resource type names follow `jamfplatform_<domain>_<entity>`:

- `jamfplatform_device_group`
- `jamfplatform_blueprints_blueprint`
- `jamfplatform_cbengine_benchmark`

### Test names

Test functions use the pattern `TestAccResource_<Resource>_<Scenario>` for acceptance tests and `Test<Function>_<Case>` for unit tests:

```go
func TestAccResource_DeviceGroup_SmartComputer(t *testing.T) { ... }
func TestAccDataSource_Baselines(t *testing.T) { ... }
func TestBuildBlueprintInput_MinimalConfig(t *testing.T) { ... }
```

### Acceptance test resource names

Use the `tf-acc-` prefix for all resources created during acceptance tests:

```
tf-acc-static-computer
tf-acc-benchmark-all-rules
tf-acc-bp-scope-passcode
```
