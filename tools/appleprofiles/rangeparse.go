// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"strconv"

	"gopkg.in/yaml.v3"
)

// parseRangelist reads Apple's `rangelist`, the closed set of values a key accepts. The element
// type is preserved rather than stringified because the wire distinguishes them: an IKEv2
// Diffie-Hellman group is the integer 14, not the string "14", and comparing an authored integer
// against a stringified table would report every valid value as out of range.
func parseRangelist(seq *yaml.Node, key string) []any {
	seq = document(seq)
	if seq == nil || seq.Kind != yaml.SequenceNode {
		return nil
	}

	values := make([]any, 0, len(seq.Content))
	for _, item := range seq.Content {
		item = document(item)
		if item == nil || item.Kind != yaml.ScalarNode {
			continue
		}
		values = append(values, scalarValue(item, key))
	}
	if len(values) == 0 {
		return nil
	}
	return values
}

// parseRange reads Apple's `range`, the inclusive bounds on a numeric key. Either bound may be
// absent — `LoginFrequency` declares only a minimum — so both are returned as pointers and a
// missing one means unbounded in that direction.
func parseRange(mapping *yaml.Node, key string) (minimum, maximum *float64) {
	mapping = document(mapping)
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return nil, nil
	}
	return numericField(mapping, "min", key), numericField(mapping, "max", key)
}

// numericField returns a mapping's numeric member, or nil when it is absent or not a number. An
// absent bound is ordinary and silent, but a bound Apple declares in a shape this cannot read is
// reported: it becomes "unbounded" in the table, so the range check for that key never runs again
// and nothing else would ever say so.
func numericField(mapping *yaml.Node, name, key string) *float64 {
	node := document(field(mapping, name))
	if node == nil {
		return nil
	}
	if node.Kind != yaml.ScalarNode {
		warnf("key %q: range %s is not a scalar (line %d); treating the bound as unbounded", key, name, node.Line)
		return nil
	}
	parsed, err := strconv.ParseFloat(node.Value, 64)
	if err != nil {
		warnf("key %q: range %s is %q, which is not a number (line %d); treating the bound as unbounded", key, name, node.Value, node.Line)
		return nil
	}
	return &parsed
}

// scalarValue converts a YAML scalar to the Go value the emitted table should carry, keyed off the
// resolved YAML tag rather than the text. Anything that is not plainly a number or a boolean stays
// a string, so an unrecognised tag degrades to the safe representation instead of being dropped.
// A value whose own tag says it is a number or a boolean and then fails to parse is reported: the
// string it degrades to can never match an authored value of the declared type, which would turn
// every valid write into a false out-of-range finding.
func scalarValue(node *yaml.Node, key string) any {
	switch node.Tag {
	case "!!int":
		if parsed, err := strconv.ParseInt(node.Value, 10, 64); err == nil {
			return parsed
		}
	case "!!float":
		if parsed, err := strconv.ParseFloat(node.Value, 64); err == nil {
			return parsed
		}
	case "!!bool":
		if parsed, err := strconv.ParseBool(node.Value); err == nil {
			return parsed
		}
	default:
		return node.Value
	}
	warnf("key %q: rangelist value %q is tagged %s but does not parse as one (line %d); keeping it as a string", key, node.Value, node.Tag, node.Line)
	return node.Value
}
