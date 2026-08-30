// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package aischemas

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/jsonvalue"
)

// walker carries the state one Validate call needs: the schema root for resolving local references,
// the findings so far, and a cache of compiled patterns so a schema that reuses one does not
// recompile it per value.
type walker struct {
	root     map[string]any
	problems []Problem
	patterns map[string]*regexp.Regexp
}

// validate checks one value against one schema node, then descends. A value whose type is already
// wrong is not descended into: the keyword checks below would all fail for the same reason, and a
// single accurate finding beats a cascade.
func (w *walker) validate(node map[string]any, value any, ptr string) {
	nodes := w.flatten(node, map[string]struct{}{}, 0)
	if len(nodes) == 0 {
		return
	}

	if !w.checkTypes(nodes, value, ptr) {
		return
	}
	w.checkEnums(nodes, value, ptr)
	w.checkBounds(nodes, value, ptr)
	w.checkBranches(nodes, value, ptr)

	switch typed := value.(type) {
	case map[string]any:
		w.validateObject(nodes, typed, ptr)
	case []any:
		w.validateArray(nodes, typed, ptr)
	}
}

// flatten expands a schema node into every node that applies to the same value, following local
// `$ref` and `allOf`. Neither keyword consumes a level of the value, so a reference cycle would not
// terminate on the value alone: visited stops a pointer being re-entered at the same value, and
// maxRefDepth stops anything the visited set cannot see.
func (w *walker) flatten(node map[string]any, visited map[string]struct{}, depth int) []map[string]any {
	if node == nil || depth > maxRefDepth {
		return nil
	}

	flattened := []map[string]any{node}

	if ref, ok := node["$ref"].(string); ok {
		if _, seen := visited[ref]; !seen {
			visited[ref] = struct{}{}
			if target, ok := w.resolve(ref); ok {
				flattened = append(flattened, w.flatten(target, visited, depth+1)...)
			}
		}
	}

	if members, ok := node["allOf"].([]any); ok {
		for _, member := range members {
			if sub, ok := member.(map[string]any); ok {
				flattened = append(flattened, w.flatten(sub, visited, depth+1)...)
			}
		}
	}
	return flattened
}

// resolve looks up a local JSON pointer reference against the schema root. Only same-document
// references are followed: every schema the service serves uses them exclusively, and following a
// remote one would mean a network fetch mid-plan.
func (w *walker) resolve(ref string) (map[string]any, bool) {
	if !strings.HasPrefix(ref, "#") {
		return nil, false
	}
	pointer := strings.TrimPrefix(ref, "#")
	if pointer == "" || pointer == "/" {
		return w.root, true
	}

	current := any(w.root)
	for segment := range strings.SplitSeq(strings.TrimPrefix(pointer, "/"), "/") {
		decoded := strings.ReplaceAll(strings.ReplaceAll(segment, "~1", "/"), "~0", "~")
		switch container := current.(type) {
		case map[string]any:
			next, ok := container[decoded]
			if !ok {
				return nil, false
			}
			current = next
		case []any:
			index, err := strconv.Atoi(decoded)
			if err != nil || index < 0 || index >= len(container) {
				return nil, false
			}
			current = container[index]
		default:
			return nil, false
		}
	}

	target, ok := current.(map[string]any)
	return target, ok
}

// checkTypes reports a value whose JSON type satisfies none of the types a node declares, and tells
// the caller whether descending is still worthwhile. A node without `type` constrains nothing.
func (w *walker) checkTypes(nodes []map[string]any, value any, ptr string) bool {
	for _, node := range nodes {
		declared := declaredTypes(node)
		if len(declared) == 0 {
			continue
		}
		if slices.ContainsFunc(declared, func(name string) bool { return matchesType(name, value) }) {
			continue
		}
		w.add(problemAt(WrongType, ptr, "expected %s, found %s.", describeTypes(declared), jsonvalue.Describe(value)))
		return false
	}
	return true
}

// checkEnums reports a value outside an enumerated set or differing from a fixed constant.
func (w *walker) checkEnums(nodes []map[string]any, value any, ptr string) {
	for _, node := range nodes {
		if allowed, ok := node["enum"].([]any); ok && !slices.ContainsFunc(allowed, func(candidate any) bool { return sameJSON(candidate, value) }) {
			w.add(problemAt(NotInEnum, ptr, "%s is not one of the accepted values: %s.", jsonvalue.Render(value), renderList(allowed)))
		}
		if fixed, ok := node["const"]; ok && !sameJSON(fixed, value) {
			w.add(problemAt(NotConst, ptr, "must be %s, found %s.", jsonvalue.Render(fixed), jsonvalue.Render(value)))
		}
	}
}

// checkBounds reports a number outside its range, a string outside its length limits or failing its
// pattern, and an array with the wrong number of entries or a repeated one.
func (w *walker) checkBounds(nodes []map[string]any, value any, ptr string) {
	for _, node := range nodes {
		if number, ok := jsonvalue.Numeric(value); ok {
			w.checkNumberBounds(node, number, ptr)
		}
		if text, ok := value.(string); ok {
			w.checkStringBounds(node, text, ptr)
		}
		if items, ok := value.([]any); ok {
			w.checkArrayBounds(node, items, ptr)
		}
	}
}

// checkNumberBounds reports a number outside the node's inclusive or exclusive bounds.
func (w *walker) checkNumberBounds(node map[string]any, number float64, ptr string) {
	if limit, ok := numberKeyword(node, "minimum"); ok && number < limit {
		w.add(problemAt(OutOfRange, ptr, "must be at least %s, found %s.", jsonvalue.FormatNumber(limit), jsonvalue.FormatNumber(number)))
	}
	if limit, ok := numberKeyword(node, "maximum"); ok && number > limit {
		w.add(problemAt(OutOfRange, ptr, "must be at most %s, found %s.", jsonvalue.FormatNumber(limit), jsonvalue.FormatNumber(number)))
	}
	if limit, ok := numberKeyword(node, "exclusiveMinimum"); ok && number <= limit {
		w.add(problemAt(OutOfRange, ptr, "must be greater than %s, found %s.", jsonvalue.FormatNumber(limit), jsonvalue.FormatNumber(number)))
	}
	if limit, ok := numberKeyword(node, "exclusiveMaximum"); ok && number >= limit {
		w.add(problemAt(OutOfRange, ptr, "must be less than %s, found %s.", jsonvalue.FormatNumber(limit), jsonvalue.FormatNumber(number)))
	}
}

// checkStringBounds reports a string outside its length limits or failing its pattern. A pattern Go
// cannot compile is skipped: draft-07 patterns are ECMA-262 and Go's regexp is RE2, so a
// lookaround a future schema introduces must not become a false finding.
func (w *walker) checkStringBounds(node map[string]any, text, ptr string) {
	length := float64(len([]rune(text)))
	if limit, ok := numberKeyword(node, "minLength"); ok && length < limit {
		w.add(problemAt(LengthOutOfRange, ptr, "must be at least %s characters, found %s.", jsonvalue.FormatNumber(limit), jsonvalue.FormatNumber(length)))
	}
	if limit, ok := numberKeyword(node, "maxLength"); ok && length > limit {
		w.add(problemAt(LengthOutOfRange, ptr, "must be at most %s characters, found %s.", jsonvalue.FormatNumber(limit), jsonvalue.FormatNumber(length)))
	}
	if pattern, ok := node["pattern"].(string); ok {
		if compiled := w.compile(pattern); compiled != nil && !compiled.MatchString(text) {
			w.add(problemAt(PatternMismatch, ptr, "%s does not match the accepted form %s.", jsonvalue.Render(text), strconv.Quote(pattern)))
		}
	}
}

// checkArrayBounds reports an array with the wrong number of entries, or a repeated entry where the
// node marks them unique.
func (w *walker) checkArrayBounds(node map[string]any, items []any, ptr string) {
	count := float64(len(items))
	if limit, ok := numberKeyword(node, "minItems"); ok && count < limit {
		w.add(problemAt(ItemCountOutOfRange, ptr, "must hold at least %s entries, found %s.", jsonvalue.FormatNumber(limit), jsonvalue.FormatNumber(count)))
	}
	if limit, ok := numberKeyword(node, "maxItems"); ok && count > limit {
		w.add(problemAt(ItemCountOutOfRange, ptr, "must hold at most %s entries, found %s.", jsonvalue.FormatNumber(limit), jsonvalue.FormatNumber(count)))
	}
	if unique, ok := node["uniqueItems"].(bool); !ok || !unique {
		return
	}
	for i := range items {
		for j := i + 1; j < len(items); j++ {
			if sameJSON(items[i], items[j]) {
				w.add(problemAt(DuplicateItems, ptr, "entries must be unique; %s appears more than once.", jsonvalue.Render(items[i])))
				return
			}
		}
	}
}

// checkBranches handles `anyOf` and `oneOf`. A branch is taken to match when it produces no
// error-level finding, so an advisory undeclared key never disqualifies one. When several match, or
// when `oneOf` matches more than one, the value is accepted: the service is the authority on
// exclusivity, and a plan-time check that guessed here would block a working configuration.
//
// When none match, the findings from the closest branch are reported alongside a summary rather than
// the findings from all of them, which for a four-way union reads as four contradictory demands.
func (w *walker) checkBranches(nodes []map[string]any, value any, ptr string) {
	for _, node := range nodes {
		for _, keyword := range []string{"anyOf", "oneOf"} {
			branches, ok := node[keyword].([]any)
			if !ok || len(branches) == 0 {
				continue
			}
			if closest, count := w.closestBranch(branches, value, ptr); count > 0 {
				w.add(problemAt(NoBranchMatches, ptr, "does not match any of the %d accepted shapes; the closest reports: %s", len(branches), closest))
			}
		}
	}
}

// closestBranch validates a value against each branch and returns the detail of the first error from
// the branch with the fewest, plus that count. A count of zero means some branch matched.
func (w *walker) closestBranch(branches []any, value any, ptr string) (string, int) {
	best := ""
	fewest := 0
	for _, branch := range branches {
		sub, ok := branch.(map[string]any)
		if !ok {
			continue
		}
		nested := &walker{root: w.root}
		nested.validate(sub, value, ptr)

		errors := 0
		detail := ""
		for _, problem := range nested.problems {
			if problem.Advisory() {
				continue
			}
			errors++
			if detail == "" {
				detail = problem.Detail
			}
		}
		if errors == 0 {
			return "", 0
		}
		if fewest == 0 || errors < fewest {
			fewest, best = errors, detail
		}
	}
	return best, fewest
}

// validateObject judges the object's keys against the union of every declared shape, then descends
// into each declared value. Judging once against the union matters: a key declared only by an `allOf`
// member is still declared, and reporting it per node would report it as undeclared.
func (w *walker) validateObject(nodes []map[string]any, value map[string]any, ptr string) {
	shape := collectObjectShape(nodes)

	for _, name := range slices.Sorted(maps.Keys(shape.required)) {
		if _, present := value[name]; !present {
			w.add(problemAt(MissingRequiredKey, ptr, "%s is required.", strconv.Quote(name)))
		}
	}

	for _, name := range slices.Sorted(maps.Keys(value)) {
		child := joinPointer(ptr, name)
		w.checkPropertyName(shape, name, child)

		declared, ok := shape.properties[name]
		if ok {
			for _, sub := range declared {
				w.validate(sub, value[name], child)
			}
			continue
		}
		w.validateUndeclared(shape, value[name], name, child, ptr)
	}
}

// checkPropertyName applies a `propertyNames` constraint to one key, which is how a map-typed object
// declares what its keys may be called.
func (w *walker) checkPropertyName(shape objectShape, name, ptr string) {
	for _, constraint := range shape.propertyNames {
		nested := &walker{root: w.root}
		nested.validate(constraint, name, ptr)
		for _, problem := range nested.problems {
			w.add(problemAt(InvalidPropertyName, ptr, "the key %s is not accepted here: %s", strconv.Quote(name), problem.Detail))
		}
	}
}

// validateUndeclared handles a key no declared shape names. A schema that supplies a schema for
// extra keys is describing a map, so the value is checked against it and the key itself is fine. A
// schema that refuses extra keys makes this an error, because the service refuses the write. A
// schema that allows them makes it advisory. A node that declares no properties at all constrains
// nothing and is passed over, which is what keeps a free-form sub-object free-form.
func (w *walker) validateUndeclared(shape objectShape, value any, name, ptr, parent string) {
	if len(shape.additionalSchemas) > 0 {
		for _, sub := range shape.additionalSchemas {
			w.validate(sub, value, ptr)
		}
		return
	}
	if !shape.declaresProperties {
		return
	}
	if shape.closed {
		w.add(problemAt(UndeclaredKey, ptr, "%s is not a setting this product accepts, and it rejects settings it does not define.", strconv.Quote(name)))
		return
	}
	w.add(problemAt(UnrecognisedKey, ptr, "%s is not a setting declared for this schema version. It will be stored and never applied unless the product added it after this schema version was published.", strconv.Quote(name)))
}

// validateArray descends into an array's entries. `items` is either one schema for every entry or a
// tuple of schemas positionally; both forms appear in the schemas the service serves.
func (w *walker) validateArray(nodes []map[string]any, items []any, ptr string) {
	for _, node := range nodes {
		switch declared := node["items"].(type) {
		case map[string]any:
			for i, item := range items {
				w.validate(declared, item, joinPointer(ptr, strconv.Itoa(i)))
			}
		case []any:
			for i, item := range items {
				if i >= len(declared) {
					break
				}
				if sub, ok := declared[i].(map[string]any); ok {
					w.validate(sub, item, joinPointer(ptr, strconv.Itoa(i)))
				}
			}
		}
	}
}

// add records a finding, skipping one already reported at the same path for the same reason. The
// union of composed nodes can restate a constraint, and a duplicated diagnostic reads as two
// separate problems.
func (w *walker) add(problem Problem) {
	if slices.ContainsFunc(w.problems, func(existing Problem) bool {
		return existing.Kind == problem.Kind && existing.Path == problem.Path && existing.Detail == problem.Detail
	}) {
		return
	}
	w.problems = append(w.problems, problem)
}

// compile returns a compiled pattern, or nil when Go's regexp cannot express it.
func (w *walker) compile(pattern string) *regexp.Regexp {
	if w.patterns == nil {
		w.patterns = map[string]*regexp.Regexp{}
	}
	if compiled, ok := w.patterns[pattern]; ok {
		return compiled
	}
	compiled, err := regexp.Compile(pattern)
	if err != nil {
		compiled = nil
	}
	w.patterns[pattern] = compiled
	return compiled
}

// objectShape is the union of what every composed node declares about an object's keys.
type objectShape struct {
	properties         map[string][]map[string]any
	required           map[string]struct{}
	additionalSchemas  []map[string]any
	propertyNames      []map[string]any
	declaresProperties bool
	closed             bool
}

// collectObjectShape merges the object keywords of every node that applies to one value.
func collectObjectShape(nodes []map[string]any) objectShape {
	shape := objectShape{properties: map[string][]map[string]any{}, required: map[string]struct{}{}}

	for _, node := range nodes {
		if declared, ok := node["properties"].(map[string]any); ok {
			shape.declaresProperties = true
			for name, sub := range declared {
				if typed, ok := sub.(map[string]any); ok {
					shape.properties[name] = append(shape.properties[name], typed)
				}
			}
		}
		if required, ok := node["required"].([]any); ok {
			for _, name := range required {
				if typed, ok := name.(string); ok {
					shape.required[typed] = struct{}{}
				}
			}
		}
		switch additional := node["additionalProperties"].(type) {
		case bool:
			if !additional {
				shape.closed = true
			}
		case map[string]any:
			shape.additionalSchemas = append(shape.additionalSchemas, additional)
		}
		if constraint, ok := node["propertyNames"].(map[string]any); ok {
			shape.propertyNames = append(shape.propertyNames, constraint)
		}
	}
	return shape
}

// declaredTypes returns the JSON types a node accepts. draft-07 allows a single name or a list.
func declaredTypes(node map[string]any) []string {
	switch declared := node["type"].(type) {
	case string:
		return []string{declared}
	case []any:
		names := make([]string, 0, len(declared))
		for _, name := range declared {
			if typed, ok := name.(string); ok {
				names = append(names, typed)
			}
		}
		return names
	default:
		return nil
	}
}

// matchesType reports whether a decoded JSON value satisfies one declared type name. `integer`
// accepts a whole-valued number, which is how JSON expresses one.
func matchesType(name string, value any) bool {
	switch name {
	case "object":
		_, ok := value.(map[string]any)
		return ok
	case "array":
		_, ok := value.([]any)
		return ok
	case "string":
		_, ok := value.(string)
		return ok
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "null":
		return value == nil
	case "number":
		_, ok := jsonvalue.Numeric(value)
		return ok
	case "integer":
		number, ok := jsonvalue.Numeric(value)
		return ok && number == float64(int64(number))
	default:
		return true
	}
}

// describeTypes names a declared type set for a diagnostic.
func describeTypes(names []string) string {
	rendered := make([]string, 0, len(names))
	for _, name := range names {
		rendered = append(rendered, jsonvalue.Article(name))
	}
	if len(rendered) == 1 {
		return rendered[0]
	}
	return strings.Join(rendered[:len(rendered)-1], ", ") + " or " + rendered[len(rendered)-1]
}

// renderList names an enumerated set for a diagnostic, truncated so a 200-value enum does not fill
// the terminal.
func renderList(values []any) string {
	const limit = 12
	rendered := make([]string, 0, min(len(values), limit))
	for _, value := range values[:min(len(values), limit)] {
		rendered = append(rendered, jsonvalue.Render(value))
	}
	joined := strings.Join(rendered, ", ")
	if len(values) > limit {
		return fmt.Sprintf("%s and %d more", joined, len(values)-limit)
	}
	return joined
}

// numberKeyword reads a numeric keyword from a node.
func numberKeyword(node map[string]any, name string) (float64, bool) {
	value, ok := node[name]
	if !ok {
		return 0, false
	}
	return jsonvalue.Numeric(value)
}

// sameJSON compares two decoded JSON values structurally, treating numbers by value so a schema's
// 1 and an author's 1.0 agree.
func sameJSON(a, b any) bool {
	if numberA, ok := jsonvalue.Numeric(a); ok {
		numberB, ok := jsonvalue.Numeric(b)
		return ok && numberA == numberB
	}
	switch typedA := a.(type) {
	case map[string]any:
		typedB, ok := b.(map[string]any)
		if !ok || len(typedA) != len(typedB) {
			return false
		}
		for name, valueA := range typedA {
			valueB, present := typedB[name]
			if !present || !sameJSON(valueA, valueB) {
				return false
			}
		}
		return true
	case []any:
		typedB, ok := b.([]any)
		if !ok || len(typedA) != len(typedB) {
			return false
		}
		for i := range typedA {
			if !sameJSON(typedA[i], typedB[i]) {
				return false
			}
		}
		return true
	default:
		return a == b
	}
}

// joinPointer appends one segment to a JSON pointer, escaping it as RFC 6901 requires.
func joinPointer(ptr, segment string) string {
	escaped := strings.ReplaceAll(strings.ReplaceAll(segment, "~", "~0"), "/", "~1")
	return ptr + "/" + escaped
}
