// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// Package appleprofiles validates configuration profile payloads against Apple's declared payload
// keys, so a payload Jamf would refuse or quietly rewrite is caught during `terraform plan` instead
// of at apply.
//
// The embedded table is generated from the `mdm/profiles` directory of apple/device-management by
// `make apple-profiles`; profiles.json records the upstream commit it was built from. Jamf's
// blueprints service validates a stored payload against the same vocabulary, which is what makes the
// table useful here — wire probing established four behaviours the checks below mirror:
//
//   - a payload type Jamf does not know is rejected outright, and the match is case-sensitive;
//   - a key Apple does not define for that payload type is silently discarded;
//   - a key that differs only in case is silently stored under Apple's spelling;
//   - a key whose value has the wrong type, or a required key left out, fails the write.
//
// Enum and range constraints are deliberately not checked: Jamf accepts values outside them
// (`AlertType: 99` stores fine), so enforcing them here would reject configurations that work.
//
// The table is a plan-time heuristic, not the authority — Jamf's own copy of Apple's schemas may lag
// or lead this snapshot. Callers must weight the findings accordingly: Kind values that report an
// unrecognised name are advisory, because a key Apple added after this snapshot would look
// unrecognised while working perfectly; the rest describe writes Jamf actively refuses.
package appleprofiles

import (
	"embed"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/jsonvalue"
)

//go:embed profiles.json
var embedded embed.FS

// Kind is the value type Apple declares for a key.
type Kind string

// The declared value types. KindAny accepts anything, and is also what the generator falls back to
// for a type it does not recognise, so an upstream vocabulary change cannot turn into a false
// finding.
const (
	KindBoolean    Kind = "boolean"
	KindInteger    Kind = "integer"
	KindReal       Kind = "real"
	KindString     Kind = "string"
	KindData       Kind = "data"
	KindDate       Kind = "date"
	KindArray      Kind = "array"
	KindDictionary Kind = "dictionary"
	KindAny        Kind = "any"
)

// Schema is the declared shape of one key's value.
type Schema struct {
	Type     Kind               `json:"type"`
	Required bool               `json:"required,omitempty"`
	Keys     map[string]*Schema `json:"keys,omitempty"`
	Any      *Schema            `json:"any,omitempty"`
	Item     *Schema            `json:"item,omitempty"`
}

// Payload is the declared shape of one payload type. Any is set when the payload accepts arbitrary
// key names.
type Payload struct {
	Title string             `json:"title,omitempty"`
	Keys  map[string]*Schema `json:"keys"`
	Any   *Schema            `json:"any,omitempty"`
}

// Table is the generated schema table.
type Table struct {
	Source   string              `json:"source"`
	Ref      string              `json:"ref"`
	Commit   string              `json:"commit"`
	Release  string              `json:"release,omitempty"`
	Payloads map[string]*Payload `json:"payloads"`
}

// load decodes the embedded table once. A table that fails to decode yields an empty one rather than
// a panic: validation is an advisory aid, and losing it must never stop a plan.
var load = sync.OnceValue(func() *Table {
	table := &Table{}
	raw, err := embedded.ReadFile("profiles.json")
	if err != nil {
		return &Table{Payloads: map[string]*Payload{}}
	}
	if err := json.Unmarshal(raw, table); err != nil {
		return &Table{Payloads: map[string]*Payload{}}
	}
	if table.Payloads == nil {
		table.Payloads = map[string]*Payload{}
	}
	return table
})

// Provenance returns the upstream commit and release the embedded table was generated from, for
// diagnostics that need to say how current the schemas are.
func Provenance() (commit, release string) {
	table := load()
	return table.Commit, table.Release
}

// PayloadTypes returns every payload type in the table, sorted.
func PayloadTypes() []string {
	return slices.Sorted(maps.Keys(load().Payloads))
}

// Lookup returns the declared shape of a payload type. The match is case-sensitive, mirroring Jamf,
// which rejects `com.apple.Dock` while accepting `com.apple.dock`.
func Lookup(payloadType string) (*Payload, bool) {
	payload, ok := load().Payloads[payloadType]
	return payload, ok
}

// ProblemKind classifies a problem found in a payload, and tells the caller how much to trust it.
type ProblemKind int

const (
	// UnknownPayloadType means the payload type is absent from the table. Advisory: Jamf may support
	// a payload type Apple does not document here.
	UnknownPayloadType ProblemKind = iota
	// MiscasedPayloadType means the payload type matches a known one apart from case. Jamf matches
	// the payload type case-sensitively and rejects the write, so this is not advisory.
	MiscasedPayloadType
	// UnknownKey means Apple does not define this key for this payload type. Advisory: a key added
	// upstream after this snapshot looks the same. Jamf discards such a key silently, so the plan
	// will never converge if the key really is unrecognised.
	UnknownKey
	// MiscasedKey means the key matches a declared key apart from case. Jamf stores it under Apple's
	// spelling, so configuration and state can never agree until the case is fixed.
	MiscasedKey
	// WrongType means the value does not match the declared type. Jamf rejects the write.
	WrongType
	// MissingRequiredKey means a key Apple marks required is absent or null. Jamf rejects the write.
	MissingRequiredKey
	// IntegerOutOfRange means an integer value does not fit the 32-bit signed field Jamf stores
	// integers in. Apple declares its integers unbounded, so this is Jamf's limit rather than
	// Apple's: wire probing found 2147483647 accepted and 2147483648 rejected across several
	// payloads (AssetCache CacheLimit, screensaver idleTime, Time Machine BackupSizeMB). Jamf
	// rejects the write, so this is not advisory.
	IntegerOutOfRange
)

// The bounds of the 32-bit signed field Jamf stores an integer payload value in.
const (
	maxJamfInteger = 2147483647
	minJamfInteger = -2147483648
)

// Problem is one finding against a payload.
type Problem struct {
	Kind ProblemKind
	// Path is the dotted path to the offending value within the payload's settings, with array
	// entries indexed (`NotificationSettings[0].BundleIdentifier`). It is empty when the problem is
	// the payload type itself.
	Path string
	// Detail describes the problem in one sentence, ready to drop into a diagnostic.
	Detail string
	// Canonical carries Apple's spelling for a miscased payload type or key, and is empty otherwise.
	Canonical string
}

// Advisory reports whether a problem rests on the table being current, rather than on behaviour
// Jamf was observed to enforce. Callers should surface an advisory problem as a warning and the rest
// as errors.
func (p Problem) Advisory() bool {
	return p.Kind == UnknownPayloadType || p.Kind == UnknownKey
}

// Validate checks one payload's settings against Apple's declared keys for its payload type,
// returning every problem found, ordered by path. A nil or empty settings map is still checked, so a
// payload that omits a required key is reported.
//
// Null values are treated as absent throughout: Jamf discards a null rather than storing it, so a
// null is never a type error, and a null under a required key is a missing key.
func Validate(payloadType string, settings map[string]any) []Problem {
	payload, ok := Lookup(payloadType)
	if !ok {
		if canonical, matched := canonicalPayloadType(payloadType); matched {
			return []Problem{{
				Kind:      MiscasedPayloadType,
				Canonical: canonical,
				Detail: fmt.Sprintf(
					"payload type %q is not spelled the way Apple defines it; Jamf matches the payload type exactly and would reject this write. Use %q.",
					payloadType, canonical,
				),
			}}
		}
		return []Problem{{
			Kind: UnknownPayloadType,
			Detail: fmt.Sprintf(
				"payload type %q is not one of the %d types Apple defines in the schemas the provider carries.",
				payloadType, len(load().Payloads),
			),
		}}
	}

	container := &Schema{Type: KindDictionary, Keys: payload.Keys, Any: payload.Any}
	var problems []Problem
	validateDictionary(container, settings, "", &problems)

	slices.SortFunc(problems, func(a, b Problem) int { return strings.Compare(a.Path, b.Path) })
	return problems
}

// canonicalPayloadType returns Apple's spelling of a payload type that matches only case-insensitively.
func canonicalPayloadType(payloadType string) (string, bool) {
	folded := strings.ToLower(payloadType)
	for _, known := range PayloadTypes() {
		if strings.ToLower(known) == folded {
			return known, true
		}
	}
	return "", false
}

// validateDictionary checks a dictionary value against its declared keys. A dictionary that accepts
// arbitrary key names is free-form: its contents are left alone entirely, because Jamf stops
// validating there too — it stores an unrecognised key inside an MCX preference domain untouched.
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
						"Apple does not define %q here; the platform discards a key it does not recognise, so it would never appear in state.",
						name,
					),
				})
				continue
			}
			*problems = append(*problems, Problem{
				Kind:      MiscasedKey,
				Path:      child,
				Canonical: canonical,
				Detail: fmt.Sprintf(
					"key %q is not spelled the way Apple defines it; the platform stores it as %q, which can never match the configuration. Use %q.",
					name, canonical, canonical,
				),
			})
			keySchema = declared.Keys[canonical]
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
		if entry, present := value[name]; present && entry != nil {
			continue
		}
		if canonical, matched := foldedPresent(value, name); matched && canonical != nil {
			continue
		}
		*problems = append(*problems, Problem{
			Kind: MissingRequiredKey,
			Path: joinPath(path, name),
			Detail: fmt.Sprintf(
				"Apple marks %q required for this payload, and the platform rejects a write that leaves it out.",
				name,
			),
		})
	}
}

// validateValue checks one value against its declared schema, descending into dictionaries and array
// entries.
func validateValue(declared *Schema, value any, path string, problems *[]Problem) {
	if declared == nil || declared.Type == KindAny {
		return
	}

	if !matchesKind(declared.Type, value) {
		*problems = append(*problems, Problem{
			Kind: WrongType,
			Path: path,
			Detail: fmt.Sprintf(
				"Apple declares %s here, but the value is %s; the platform rejects a write whose value has the wrong type.",
				jsonvalue.Article(string(declared.Type)), jsonvalue.Describe(value),
			),
		})
		return
	}

	if declared.Type == KindInteger {
		if number, ok := jsonvalue.Numeric(value); ok && (number > maxJamfInteger || number < minJamfInteger) {
			*problems = append(*problems, Problem{
				Kind: IntegerOutOfRange,
				Path: path,
				Detail: fmt.Sprintf(
					"Apple leaves this integer unbounded, but the platform stores it in a 32-bit signed field and rejects a write outside %d to %d; the value is %s.",
					minJamfInteger, maxJamfInteger, jsonvalue.FormatNumber(number),
				),
			})
		}
	}

	switch declared.Type {
	case KindDictionary:
		nested, ok := value.(map[string]any)
		if !ok {
			return
		}
		validateDictionary(declared, nested, path, problems)
	case KindArray:
		entries, ok := value.([]any)
		if !ok || declared.Item == nil {
			return
		}
		for i, entry := range entries {
			if entry == nil {
				continue
			}
			validateValue(declared.Item, entry, fmt.Sprintf("%s[%d]", path, i), problems)
		}
	}
}

// matchesKind reports whether a decoded JSON value matches a declared type. Numbers arrive as
// float64 from encoding/json, so an integer is a float64 with no fractional part; `data` and `date`
// are carried as strings in JSON. Whole-ness is judged by value rather than by a round-trip through
// int64, so a number beyond int64 stays an integer here and is reported by the 32-bit range check
// below, which is the problem an author actually has.
func matchesKind(declared Kind, value any) bool {
	switch declared {
	case KindAny:
		return true
	case KindBoolean:
		_, ok := value.(bool)
		return ok
	case KindInteger:
		number, ok := jsonvalue.Numeric(value)
		return ok && jsonvalue.IsWhole(number)
	case KindReal:
		_, ok := jsonvalue.Numeric(value)
		return ok
	case KindString, KindData, KindDate:
		_, ok := value.(string)
		return ok
	case KindArray:
		_, ok := value.([]any)
		return ok
	case KindDictionary:
		_, ok := value.(map[string]any)
		return ok
	default:
		return true
	}
}

// foldedIndex maps each declared key to itself by lower-cased name, for case-insensitive lookup.
func foldedIndex(keys map[string]*Schema) map[string]string {
	index := make(map[string]string, len(keys))
	for name := range keys {
		index[strings.ToLower(name)] = name
	}
	return index
}

// foldedPresent reports whether a value map carries a key under a different case, so a required key
// the author miscased is reported once, as a case problem, rather than twice.
func foldedPresent(value map[string]any, name string) (any, bool) {
	folded := strings.ToLower(name)
	for candidate, entry := range value {
		if strings.ToLower(candidate) == folded {
			return entry, true
		}
	}
	return nil, false
}

// joinPath appends a key to a dotted path.
func joinPath(path, name string) string {
	if path == "" {
		return name
	}
	return path + "." + name
}
