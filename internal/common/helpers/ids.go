// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package helpers

import (
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// IntIDToString converts a Jamf integer ID into a Terraform string attribute.
func IntIDToString(id int64) types.String {
	return types.StringValue(strconv.FormatInt(id, 10))
}

// StringToIntID parses a Terraform string ID into an int64 for SDK calls.
func StringToIntID(s types.String) (int64, error) {
	if s.IsNull() || s.IsUnknown() {
		return 0, fmt.Errorf("id is null or unknown")
	}
	v, err := strconv.ParseInt(s.ValueString(), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("id %q is not a valid integer: %w", s.ValueString(), err)
	}
	return v, nil
}
