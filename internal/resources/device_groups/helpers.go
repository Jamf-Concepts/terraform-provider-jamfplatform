// Copyright 2025 Jamf Software LLC.

package device_groups

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// buildDeviceGroupFilter constructs the API filter string from configured filter blocks.
func buildDeviceGroupFilter(data *DeviceGroupsDataSourceModel) string {
	clauses := []string{}
	joiners := []string{}
	for _, filter := range data.Filters {
		selector, hasSelector := configuredFilterValue(filter.Selector)
		argument, hasArgument := configuredFilterValue(filter.Argument)
		if !hasSelector || !hasArgument {
			continue
		}
		if !isValidDeviceGroupSelector(selector) {
			continue
		}

		operator := "=="
		if value, ok := configuredFilterValue(filter.Operator); ok {
			operator = value
		}

		clause := fmt.Sprintf("%s%s%s", selector, operator, formatArgument(argument))
		clauses = append(clauses, clause)

		joinWith := "and"
		if value, ok := configuredFilterValue(filter.JoinWith); ok {
			logic := strings.ToLower(value)
			if logic == "or" {
				joinWith = "or"
			}
		}
		joiners = append(joiners, joinWith)
	}
	if len(clauses) == 0 {
		return ""
	}
	var builder strings.Builder
	builder.WriteString(clauses[0])
	for i := 1; i < len(clauses); i++ {
		logic := "and"
		if i < len(joiners) {
			logic = joiners[i]
		}
		builder.WriteString(" ")
		builder.WriteString(logic)
		builder.WriteString(" ")
		builder.WriteString(clauses[i])
	}
	return builder.String()
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

func isValidDeviceGroupSelector(selector string) bool {
	switch selector {
	case "name", "description", "deviceType", "groupType":
		return true
	default:
		return false
	}
}

// formatArgument ensures an argument is properly escaped and quoted when required by RSQL.
func formatArgument(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if alreadyWrappedArgument(trimmed) || looksLikeListArgument(trimmed) {
		return trimmed
	}
	escaped := strings.ReplaceAll(trimmed, "\"", "\\\"")
	if argumentNeedsQuoting(trimmed) {
		return fmt.Sprintf("\"%s\"", escaped)
	}
	return escaped
}

func alreadyWrappedArgument(value string) bool {
	if len(value) < 2 {
		return false
	}
	return (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
		(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'"))
}

func looksLikeListArgument(value string) bool {
	return strings.HasPrefix(value, "(") && strings.HasSuffix(value, ")")
}

func argumentNeedsQuoting(value string) bool {
	return strings.ContainsAny(value, " ,;()\t")
}
