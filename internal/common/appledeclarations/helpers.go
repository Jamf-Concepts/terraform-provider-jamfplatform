// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package appledeclarations

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// foldedIndex maps each declared key's lower-cased name to Apple's spelling, so a key that differs
// only in case can be named in the diagnostic rather than reported as unknown.
func foldedIndex(keys map[string]*Schema) map[string]string {
	index := make(map[string]string, len(keys))
	for name := range keys {
		index[strings.ToLower(name)] = name
	}
	return index
}

// joinPath appends a key to a dotted path, so the diagnostic points at the offending value rather
// than at the declaration as a whole.
func joinPath(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + "." + name
}

// numeric returns a value's numeric content. JSON decoding yields float64 for every number, but the
// int cases are handled too so a caller building a payload in Go rather than decoding one is not
// reported as having the wrong type.
func numeric(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case int64:
		return float64(typed), true
	default:
		return 0, false
	}
}

// matchesEnum reports whether a value equals one of the values Apple declares. Numbers are compared
// numerically rather than by representation: Apple declares an IKEv2 Diffie-Hellman group as the
// integer 14, JSON decoding yields float64(14), and comparing those as strings would reject every
// valid value.
func matchesEnum(value any, allowed []any) bool {
	if number, ok := numeric(value); ok {
		for _, candidate := range allowed {
			if other, ok := numeric(candidate); ok && other == number {
				return true
			}
		}
		return false
	}
	return slices.Contains(allowed, value)
}

// render describes a value for a diagnostic: the value itself when it is short and printable, and
// its type when spelling it out would bury the message.
func render(value any) string {
	switch typed := value.(type) {
	case nil:
		return "null"
	case string:
		if len(typed) > 60 {
			return "a string"
		}
		return strconv.Quote(typed)
	case bool:
		return strconv.FormatBool(typed)
	case map[string]any:
		return "an object"
	case []any:
		return "an array"
	default:
		if number, ok := numeric(value); ok {
			return renderNumber(number)
		}
		return fmt.Sprintf("%v", value)
	}
}

// renderNumber prints a number without a trailing decimal point for integral values, so a range
// diagnostic reads "999" rather than "999.000000".
func renderNumber(number float64) string {
	return strconv.FormatFloat(number, 'f', -1, 64)
}

// renderList prints Apple's declared value set for a diagnostic.
func renderList(values []any) string {
	rendered := make([]string, 0, len(values))
	for _, value := range values {
		rendered = append(rendered, render(value))
	}
	return strings.Join(rendered, ", ")
}

// renderBounds prints a declared range, naming only the bound that exists — Apple declares some
// keys with a minimum and no maximum.
func renderBounds(minimum, maximum *float64) string {
	switch {
	case minimum != nil && maximum != nil:
		return renderNumber(*minimum) + " to " + renderNumber(*maximum)
	case minimum != nil:
		return renderNumber(*minimum) + " or greater"
	case maximum != nil:
		return renderNumber(*maximum) + " or less"
	default:
		return "no declared bounds"
	}
}
