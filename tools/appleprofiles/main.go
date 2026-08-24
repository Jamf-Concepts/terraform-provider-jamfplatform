// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// Command appleprofiles converts Apple's configuration profile schemas into the compact JSON table
// the provider embeds at internal/common/appleprofiles/profiles.json.
//
// The source is the `mdm/profiles` directory of apple/device-management, which declares, per payload
// type, every key Apple defines and its value type. Jamf's blueprints service validates a legacy
// payload against the same vocabulary, so the table lets the provider catch at plan time what would
// otherwise be a silently discarded key or a failed apply.
//
// Run it through `make apple-profiles`, which pins the upstream checkout and passes the resolved
// commit so the generated table records exactly what it was built from.
//
// Usage:
//
//	go run ./appleprofiles -source <checkout>/mdm/profiles -commit <sha> -release <tag> -out <path>
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// wildcardKey is the key name Apple uses for a dictionary that accepts arbitrary key names — an app
// preference domain under com.apple.ManagedClient.preferences, for instance. Everything below one is
// free-form, and the generator stops descending there: Jamf stops validating there too, so a schema
// carried past this point would only manufacture findings the service does not agree with.
const wildcardKey = "ANY"

// maxDepth bounds descent. Two upstream schemas define recursive structures through YAML anchors
// (com.apple.applicationaccess.new, com.apple.homescreenlayout); beyond this depth the subtree is
// recorded as free-form rather than expanded forever.
const maxDepth = 12

// excludedPayloads are the upstream files that do not describe a payload type an author can write.
// CommonPayloadKeys holds the metadata keys every payload carries — those are merged into each
// payload instead (see commonKeys). TopLevel describes the .mobileconfig envelope around payloads,
// not a payload.
var excludedPayloads = map[string]bool{
	"CommonPayloadKeys": true,
	"TopLevel":          true,
}

// specKey is one entry of a payload's `payloadkeys` list, or of a nested `subkeys` list.
type specKey struct {
	Key      string
	Type     string
	Presence string
	Subkeys  []*specKey
}

// specFile is one upstream payload schema file.
type specFile struct {
	Title       string
	PayloadType string
	PayloadKeys []*specKey
}

// parseSpec reads one upstream schema file. It walks the YAML at node level rather than unmarshalling
// into structs, because two upstream schemas define a structure that contains itself through a YAML
// anchor (com.apple.applicationaccess.new, com.apple.homescreenlayout) and struct unmarshalling
// rejects those outright. Walking nodes lets the recursion be cut where it closes.
func parseSpec(raw []byte) (*specFile, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	root := document(&doc)
	if root == nil {
		return nil, fmt.Errorf("empty document")
	}

	spec := &specFile{
		Title:       scalar(field(root, "title")),
		PayloadType: scalar(field(document(field(root, "payload")), "payloadtype")),
	}
	spec.PayloadKeys = parseKeys(field(root, "payloadkeys"), map[*yaml.Node]bool{})
	return spec, nil
}

// parseKeys converts a `payloadkeys` or `subkeys` sequence into spec keys. visiting holds the
// sequence nodes currently on the stack, so a sequence that reaches itself through an alias is cut
// rather than expanded forever; the affected key becomes a free-form dictionary.
func parseKeys(seq *yaml.Node, visiting map[*yaml.Node]bool) []*specKey {
	seq = document(seq)
	if seq == nil || seq.Kind != yaml.SequenceNode || visiting[seq] {
		return nil
	}
	visiting[seq] = true
	defer delete(visiting, seq)

	keys := make([]*specKey, 0, len(seq.Content))
	for _, item := range seq.Content {
		item = document(item)
		if item == nil || item.Kind != yaml.MappingNode {
			continue
		}
		keys = append(keys, &specKey{
			Key:      scalar(field(item, "key")),
			Type:     scalar(field(item, "type")),
			Presence: scalar(field(item, "presence")),
			Subkeys:  parseKeys(field(item, "subkeys"), visiting),
		})
	}
	return keys
}

// document follows a document wrapper and resolves an alias to the node it points at, so callers
// always see the mapping, sequence, or scalar itself.
func document(node *yaml.Node) *yaml.Node {
	for node != nil {
		switch {
		case node.Kind == yaml.DocumentNode && len(node.Content) > 0:
			node = node.Content[0]
		case node.Kind == yaml.AliasNode:
			node = node.Alias
		default:
			return node
		}
	}
	return nil
}

// field returns the value node of a mapping key, or nil when the mapping has no such key.
func field(mapping *yaml.Node, name string) *yaml.Node {
	mapping = document(mapping)
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == name {
			return mapping.Content[i+1]
		}
	}
	return nil
}

// scalar returns a node's string value, or "" when the node is absent or not a scalar.
func scalar(node *yaml.Node) string {
	node = document(node)
	if node == nil || node.Kind != yaml.ScalarNode {
		return ""
	}
	return node.Value
}

// schema is the emitted shape of one key's value. Exactly one of Keys/Any/Item is populated,
// according to Type.
type schema struct {
	Type     string             `json:"type"`
	Required bool               `json:"required,omitempty"`
	Keys     map[string]*schema `json:"keys,omitempty"`
	Any      *schema            `json:"any,omitempty"`
	Item     *schema            `json:"item,omitempty"`
}

// payload is the emitted shape of one payload type. Any is set for the handful of payloads whose
// own key list is a wildcard (the ethernet.managed family), meaning any key name is accepted.
type payload struct {
	Title string             `json:"title,omitempty"`
	Keys  map[string]*schema `json:"keys"`
	Any   *schema            `json:"any,omitempty"`
}

// table is the emitted table, written to profiles.json.
type table struct {
	Source   string              `json:"source"`
	Ref      string              `json:"ref"`
	Commit   string              `json:"commit"`
	Release  string              `json:"release,omitempty"`
	Payloads map[string]*payload `json:"payloads"`
}

func main() {
	source := flag.String("source", "", "path to the mdm/profiles directory of an apple/device-management checkout")
	commit := flag.String("commit", "", "upstream commit the checkout is at")
	release := flag.String("release", "", "upstream release tag or commit subject")
	ref := flag.String("ref", "release", "upstream branch")
	out := flag.String("out", "", "path to write the generated JSON table to")
	flag.Parse()

	if *source == "" || *out == "" || *commit == "" {
		fmt.Fprintln(os.Stderr, "appleprofiles: -source, -commit and -out are required")
		os.Exit(2)
	}

	generated, err := generate(*source, *commit, *release, *ref)
	if err != nil {
		fmt.Fprintln(os.Stderr, "appleprofiles:", err)
		os.Exit(1)
	}

	encoded, err := json.MarshalIndent(generated, "", " ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "appleprofiles:", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*out, append(encoded, '\n'), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "appleprofiles:", err)
		os.Exit(1)
	}

	fmt.Printf("appleprofiles: wrote %d payload types to %s (upstream %s)\n", len(generated.Payloads), *out, *commit)
}

// generate reads every payload schema in the source directory and reduces it to the emitted table.
func generate(source, commit, release, ref string) (*table, error) {
	entries, err := filepath.Glob(filepath.Join(source, "*.yaml"))
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("no schema files found under %s", source)
	}
	sort.Strings(entries)

	common, err := commonKeys(source)
	if err != nil {
		return nil, err
	}

	generated := &table{
		Source:   "https://github.com/apple/device-management",
		Ref:      ref,
		Commit:   commit,
		Release:  release,
		Payloads: make(map[string]*payload, len(entries)),
	}

	for _, path := range entries {
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		spec, err := parseSpec(raw)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", filepath.Base(path), err)
		}

		payloadType := strings.TrimSpace(spec.PayloadType)
		if payloadType == "" || excludedPayloads[payloadType] {
			continue
		}

		// Several MCX files share one payload type (com.apple.MCX), each describing a different
		// preference domain, so their key sets merge into a single entry.
		entry := generated.Payloads[payloadType]
		if entry == nil {
			entry = &payload{Title: spec.Title, Keys: make(map[string]*schema)}
			generated.Payloads[payloadType] = entry
		}

		for name, keySchema := range dictionaryKeys(spec.PayloadKeys, 1) {
			entry.Keys[name] = keySchema
		}
		for _, key := range spec.PayloadKeys {
			if key != nil && key.Key == wildcardKey {
				entry.Any = reduce(key, 1)
				entry.Any.Required = false
			}
		}
		for name, keySchema := range common {
			if _, exists := entry.Keys[name]; !exists {
				entry.Keys[name] = keySchema
			}
		}
	}

	return generated, nil
}

// commonKeys reads CommonPayloadKeys.yaml, the metadata keys every payload carries. They are merged
// into each payload type because the service accepts them inside a payload and echoes an authored
// value back, so an author who writes one must not see it reported as unknown. Their `required`
// presence is dropped: the provider supplies the identity keys itself, and the service stamps the
// rest, so demanding them from an author's settings would be wrong.
func commonKeys(source string) (map[string]*schema, error) {
	raw, err := os.ReadFile(filepath.Join(source, "CommonPayloadKeys.yaml"))
	if err != nil {
		return nil, err
	}

	spec, err := parseSpec(raw)
	if err != nil {
		return nil, fmt.Errorf("CommonPayloadKeys.yaml: %w", err)
	}

	keys := dictionaryKeys(spec.PayloadKeys, 1)
	for _, keySchema := range keys {
		keySchema.Required = false
	}
	return keys, nil
}

// dictionaryKeys reduces a list of spec keys to the emitted key map, dropping the wildcard entry —
// the caller records that separately, since it applies to any key name rather than one.
func dictionaryKeys(keys []*specKey, depth int) map[string]*schema {
	reduced := make(map[string]*schema, len(keys))
	for _, key := range keys {
		if key == nil || key.Key == "" || key.Key == wildcardKey {
			continue
		}
		reduced[key.Key] = reduce(key, depth)
	}
	return reduced
}

// reduce converts one spec key to its emitted schema, descending into nested dictionaries and array
// element types. An array's element schema is its single subkey: upstream names that subkey for
// documentation (`PayloadContentItem`, `Settings`), but a JSON array is positional, so the name is
// dropped and only the shape is kept.
func reduce(key *specKey, depth int) *schema {
	kind := normaliseType(key.Type)
	result := &schema{Type: kind, Required: key.Presence == "required"}

	if depth >= maxDepth {
		result.Type = typeAny
		return result
	}

	switch kind {
	case typeDictionary:
		for _, sub := range key.Subkeys {
			if sub != nil && sub.Key == wildcardKey {
				result.Any = reduce(sub, depth+1)
				result.Any.Required = false
			}
		}
		if named := dictionaryKeys(key.Subkeys, depth+1); len(named) > 0 {
			result.Keys = named
		}
	case typeArray:
		for _, sub := range key.Subkeys {
			if sub == nil {
				continue
			}
			result.Item = reduce(sub, depth+1)
			result.Item.Required = false
			break
		}
	}

	return result
}

// The emitted type vocabulary, mapping Apple's `<...>` spelling to a bare token.
const (
	typeAny        = "any"
	typeArray      = "array"
	typeDictionary = "dictionary"
)

// normaliseType maps Apple's declared type to the emitted token, falling back to the permissive
// `any` for anything unrecognised so a vocabulary change upstream cannot turn into a false finding.
func normaliseType(declared string) string {
	switch strings.ToLower(strings.Trim(declared, "<>")) {
	case "boolean":
		return "boolean"
	case "integer":
		return "integer"
	case "real":
		return "real"
	case "string":
		return "string"
	case "data":
		return "data"
	case "date":
		return "date"
	case "array":
		return typeArray
	case "dictionary":
		return typeDictionary
	default:
		return typeAny
	}
}
