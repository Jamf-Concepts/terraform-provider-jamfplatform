// Copyright 2025 Jamf Software LLC.

package benchmarks

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// stringValueOrNull converts an empty string into a Terraform null for easier schema population.
func stringValueOrNull(value string) types.String {
	if value == "" {
		return types.StringNull()
	}
	return types.StringValue(value)
}
