package benchmark

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// SingleItemListValidator ensures a list contains only one item.
type SingleItemListValidator struct{}

// Description returns a human-readable description of the validator.
func (v SingleItemListValidator) Description(_ context.Context) string {
	return "Ensures the list contains exactly one item."
}

// MarkdownDescription returns a markdown-formatted description of the validator.
func (v SingleItemListValidator) MarkdownDescription(_ context.Context) string {
	return "Ensures the list contains exactly one item."
}

// ValidateList validates the list to ensure it contains exactly one item.
func (v SingleItemListValidator) ValidateList(ctx context.Context, req validator.ListRequest, resp *validator.ListResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	var items []string
	diags := req.ConfigValue.ElementsAs(ctx, &items, false)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	if len(items) != 1 {
		resp.Diagnostics.AddError(
			"Invalid number of items",
			fmt.Sprintf("Attribute '%s' must have exactly 1 item, but %d were provided.", req.Path.String(), len(items)),
		)
	}
}

// SingleItemList returns a validator.List that ensures a list contains exactly one item.
func SingleItemList() validator.List {
	return SingleItemListValidator{}
}
