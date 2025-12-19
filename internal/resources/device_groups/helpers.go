// Copyright 2025 Jamf Software LLC.

package device_groups

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// buildDeviceGroupFilter constructs the API filter string based on configured data source attributes.
func buildDeviceGroupFilter(data *DeviceGroupsDataSourceModel) string {
	clauses := []string{}
	if value, ok := configuredFilterValue(data.Name); ok {
		clauses = append(clauses, fmt.Sprintf(`name=="%s"`, escapeFilterValue(value)))
	}
	if value, ok := configuredFilterValue(data.Description); ok {
		clauses = append(clauses, fmt.Sprintf(`description=="%s"`, escapeFilterValue(value)))
	}
	if value, ok := configuredFilterValue(data.DeviceType); ok {
		clauses = append(clauses, fmt.Sprintf(`deviceType=="%s"`, escapeFilterValue(strings.ToUpper(value))))
	}
	if value, ok := configuredFilterValue(data.GroupType); ok {
		clauses = append(clauses, fmt.Sprintf(`groupType=="%s"`, escapeFilterValue(strings.ToUpper(value))))
	}
	return strings.Join(clauses, " and ")
}

// configuredFilterValue checks if a types.String is set and returns its value.
func configuredFilterValue(value types.String) (string, bool) {
	if value.IsNull() || value.IsUnknown() {
		return "", false
	}
	str := value.ValueString()
	if str == "" {
		return "", false
	}
	return str, true
}

// escapeFilterValue escapes double quotes in filter values for API queries.
func escapeFilterValue(value string) string {
	return strings.ReplaceAll(value, "\"", `\\"`)
}
