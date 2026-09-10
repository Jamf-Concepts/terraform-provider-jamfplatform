// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package appledeclarations

import (
	"fmt"
	"maps"
	"math"
	"slices"
	"strings"
)

// ProblemKind classifies a finding against a declaration. Every kind is an error: each one names
// behaviour the service or the device was observed to reject or silently discard, so none of them
// can produce a declaration that works. See the package doc for the wire evidence behind each.
type ProblemKind int

const (
	// UnknownDeclarationType means Apple declares no such type on either branch the table unions.
	// The service stores it and the editor renders no card, so it never reaches a device.
	UnknownDeclarationType ProblemKind = iota
	// MiscasedDeclarationType means the type matches a declared one apart from case. The service
	// matches case-sensitively, so this behaves exactly like an unknown type.
	MiscasedDeclarationType
	// KindMismatch means `kind` disagrees with the type's reverse-domain prefix. The service
	// accepts any pairing and passes the mismatch through to the device.
	KindMismatch
	// UnknownKey means Apple declares no such key for this declaration type. The service discards
	// an unrecognised key silently, so it can never appear in state or on the device.
	UnknownKey
	// MiscasedKey means the key matches a declared key apart from case. Unlike a configuration
	// profile payload, where Jamf restores Apple's spelling, a declaration key is DISCARDED.
	MiscasedKey
	// WrongType means the value does not match the declared type. The key is recognised but the
	// value is dropped, leaving the setting unapplied.
	WrongType
	// MissingRequiredKey means a key Apple marks required is absent or null.
	MissingRequiredKey
	// NotInEnum means the value is outside the closed set Apple declares in `rangelist`. The
	// service recognises the key and drops the value.
	NotInEnum
	// OutOfRange means a numeric value falls outside Apple's declared `range`. Jamf does NOT
	// enforce this — it stores and renders the value unchanged — but the device rejects it, so the
	// declaration silently never applies.
	OutOfRange
	// UnknownStatusItem means a status subscription names an item Apple does not publish. Nothing
	// in Apple's schemas types these strings, so this is the only check that can catch a typo.
	UnknownStatusItem
)

// String names the kind, for test failure messages and diagnostics that group findings.
func (k ProblemKind) String() string {
	switch k {
	case UnknownDeclarationType:
		return "unknown declaration type"
	case MiscasedDeclarationType:
		return "miscased declaration type"
	case KindMismatch:
		return "kind mismatch"
	case UnknownKey:
		return "unknown key"
	case MiscasedKey:
		return "miscased key"
	case WrongType:
		return "wrong type"
	case MissingRequiredKey:
		return "missing required key"
	case NotInEnum:
		return "value not in enum"
	case OutOfRange:
		return "value out of range"
	case UnknownStatusItem:
		return "unknown status item"
	default:
		return "unknown problem"
	}
}

// Problem is one finding against a declaration.
type Problem struct {
	Kind ProblemKind
	// Path is the dotted path to the offending value within the declaration's payload, with array
	// entries indexed (`StatusItems[0].Name`). It is empty when the problem is the declaration
	// type or its kind rather than something inside the payload.
	Path string
	// Detail describes the problem in one sentence, ready to drop into a diagnostic.
	Detail string
	// Canonical carries Apple's spelling for a miscased type or key, and is empty otherwise.
	Canonical string
	// SeedOnly marks a finding about a key or type that the table knows only from Apple's
	// pre-release branch. It does not soften the finding — the value is still wrong — but it tells
	// the operator the surrounding schema is newer than the released one.
	SeedOnly bool
}

// Validate checks one declaration against Apple's schemas, returning every problem found, ordered
// by path. A nil payload is still checked, so a declaration omitting a required key is reported.
//
// Null values are treated as absent throughout: a null is never a type error, and a null under a
// required key is a missing key.
func Validate(kind, declarationType string, payload map[string]any) []Problem {
	declaration, ok := Lookup(declarationType)
	if !ok {
		if canonical, matched := canonicalType(declarationType); matched {
			return []Problem{{
				Kind:      MiscasedDeclarationType,
				Canonical: canonical,
				Detail: fmt.Sprintf(
					"declaration type %q is not spelled the way Apple defines it. The platform matches the type exactly and would deliver nothing. Use %q.",
					declarationType, canonical,
				),
			}}
		}
		return []Problem{{
			Kind: UnknownDeclarationType,
			Detail: fmt.Sprintf(
				"declaration type %q is not one of the %d types Apple declares in the schemas the provider carries (%s). The platform stores an unrecognised type and delivers nothing.",
				declarationType, len(load().Declarations), strings.Join(Refs(), " + "),
			),
		}}
	}

	var problems []Problem

	if expected := KindForType(declarationType); expected != "" && kind != "" && kind != expected {
		problems = append(problems, Problem{
			Kind: KindMismatch,
			Detail: fmt.Sprintf(
				"declaration type %q is a %s declaration, but kind is %q. The platform accepts any pairing and passes the mismatch to the device.",
				declarationType, expected, kind,
			),
			SeedOnly: declaration.SeedOnly(),
		})
	}

	container := &Schema{Type: KindDictionary, Keys: declaration.Keys, Any: declaration.Any}
	validateDictionary(container, payload, "", &problems)

	if declarationType == statusSubscriptionsType {
		validateStatusSubscriptions(payload, &problems)
	}

	slices.SortFunc(problems, func(a, b Problem) int {
		if cmp := strings.Compare(a.Path, b.Path); cmp != 0 {
			return cmp
		}
		return int(a.Kind) - int(b.Kind)
	})
	return problems
}

// canonicalType returns Apple's spelling of a declaration type that matches only case-insensitively.
func canonicalType(declarationType string) (string, bool) {
	folded := strings.ToLower(declarationType)
	for _, known := range DeclarationTypes() {
		if strings.ToLower(known) == folded {
			return known, true
		}
	}
	return "", false
}

// validateDictionary checks a dictionary value against its declared keys. A dictionary accepting
// arbitrary key names is free-form and its contents are left alone, because Apple declares no
// vocabulary to check them against.
func validateDictionary(declared *Schema, value map[string]any, path string, problems *[]Problem) {
	if declared.Any != nil {
		return
	}

	index := foldedIndex(declared.Keys)
	for _, name := range slices.Sorted(maps.Keys(value)) {
		entry := value[name]
		child := joinPath(path, name)

		keySchema, exact := declared.Keys[name]
		if !exact {
			canonical, matched := index[strings.ToLower(name)]
			if !matched {
				*problems = append(*problems, Problem{
					Kind: UnknownKey,
					Path: child,
					Detail: fmt.Sprintf(
						"Apple does not declare %q here. The platform discards a key it does not recognise, so this setting would never reach a device.",
						name,
					),
				})
				continue
			}
			*problems = append(*problems, Problem{
				Kind:      MiscasedKey,
				Path:      child,
				Canonical: canonical,
				SeedOnly:  declared.Keys[canonical].SeedOnly(),
				Detail: fmt.Sprintf(
					"key %q is not spelled the way Apple declares it. Declaration keys are matched exactly and a wrong-cased key is discarded, unlike a configuration profile payload. Use %q.",
					name, canonical,
				),
			})
			continue
		}

		if entry == nil {
			continue
		}
		validateValue(keySchema, entry, child, problems)
	}

	for _, name := range slices.Sorted(maps.Keys(declared.Keys)) {
		keySchema := declared.Keys[name]
		if !keySchema.Required {
			continue
		}
		if entry, present := value[name]; !present || entry == nil {
			*problems = append(*problems, Problem{
				Kind:     MissingRequiredKey,
				Path:     joinPath(path, name),
				SeedOnly: keySchema.SeedOnly(),
				Detail: fmt.Sprintf(
					"Apple marks %q required here, and it is absent. The device rejects a declaration missing a required key.",
					name,
				),
			})
		}
	}
}

// validateValue checks one value against its declared schema, descending into dictionaries and
// array elements.
func validateValue(declared *Schema, value any, path string, problems *[]Problem) {
	if declared == nil || declared.Type == KindAny {
		return
	}

	switch declared.Type {
	case KindDictionary:
		nested, ok := value.(map[string]any)
		if !ok {
			appendWrongType(problems, declared, path, "an object", value)
			return
		}
		validateDictionary(declared, nested, path, problems)
		return
	case KindArray:
		items, ok := value.([]any)
		if !ok {
			appendWrongType(problems, declared, path, "an array", value)
			return
		}
		for i, item := range items {
			if item == nil {
				continue
			}
			validateValue(declared.Item, item, fmt.Sprintf("%s[%d]", path, i), problems)
		}
		return
	case KindBoolean:
		if _, ok := value.(bool); !ok {
			appendWrongType(problems, declared, path, "a boolean", value)
			return
		}
	case KindInteger:
		number, ok := numeric(value)
		if !ok {
			appendWrongType(problems, declared, path, "an integer", value)
			return
		}
		if number != math.Trunc(number) {
			appendWrongType(problems, declared, path, "an integer", value)
			return
		}
		checkRange(problems, declared, path, number)
	case KindReal:
		number, ok := numeric(value)
		if !ok {
			appendWrongType(problems, declared, path, "a number", value)
			return
		}
		checkRange(problems, declared, path, number)
	case KindString, KindData, KindDate:
		if _, ok := value.(string); !ok {
			appendWrongType(problems, declared, path, "a string", value)
			return
		}
	}

	checkEnum(problems, declared, path, value)
}

// checkEnum reports a value outside the closed set Apple declares for a key.
func checkEnum(problems *[]Problem, declared *Schema, path string, value any) {
	if len(declared.Enum) == 0 || matchesEnum(value, declared.Enum) {
		return
	}
	*problems = append(*problems, Problem{
		Kind:     NotInEnum,
		Path:     path,
		SeedOnly: declared.SeedOnly(),
		Detail: fmt.Sprintf(
			"%s is not one of the values Apple declares here (%s). The platform recognises the key and drops a value outside that set.",
			render(value), renderList(declared.Enum),
		),
	})
}

// checkRange reports a numeric value outside Apple's declared bounds. Jamf stores such a value
// unchanged, so the rejection happens on the device — which is why this is an error and not a note.
func checkRange(problems *[]Problem, declared *Schema, path string, number float64) {
	below := declared.Min != nil && number < *declared.Min
	above := declared.Max != nil && number > *declared.Max
	if !below && !above {
		return
	}
	*problems = append(*problems, Problem{
		Kind:     OutOfRange,
		Path:     path,
		SeedOnly: declared.SeedOnly(),
		Detail: fmt.Sprintf(
			"%s is outside the range Apple declares here (%s). Jamf stores an out-of-range value unchanged, so the device is what rejects it and the setting silently never applies.",
			render(number), renderBounds(declared.Min, declared.Max),
		),
	})
}

// validateStatusSubscriptions checks the names a status-subscriptions declaration subscribes to
// against the vocabulary Apple publishes. Apple types these as plain strings, so nothing else in
// the schema can catch a typo here.
func validateStatusSubscriptions(payload map[string]any, problems *[]Problem) {
	known := load().StatusItems
	if len(known) == 0 {
		return
	}
	items, ok := payload["StatusItems"].([]any)
	if !ok {
		return
	}
	for i, entry := range items {
		item, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		name, ok := item["Name"].(string)
		if !ok || slices.Contains(known, name) {
			continue
		}
		*problems = append(*problems, Problem{
			Kind: UnknownStatusItem,
			Path: fmt.Sprintf("StatusItems[%d].Name", i),
			Detail: fmt.Sprintf(
				"%q is not a status item Apple publishes. A subscription to an unpublished item never reports.",
				name,
			),
		})
	}
}

// appendWrongType records a value whose type does not match the declaration.
func appendWrongType(problems *[]Problem, declared *Schema, path, expected string, value any) {
	*problems = append(*problems, Problem{
		Kind:     WrongType,
		Path:     path,
		SeedOnly: declared.SeedOnly(),
		Detail: fmt.Sprintf(
			"Apple declares this as %s, and %s was given. The platform recognises the key and drops a value of the wrong type.",
			expected, render(value),
		),
	})
}
