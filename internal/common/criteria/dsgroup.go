// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package criteria

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/ldapgroups"
)

// Directory-service group criteria (Jamf Pro 11.29+) carry a base64-encoded JSON
// object — {"uuid":"…","serverId":"…"} — as their criterion VALUE, not a group
// name. This file maps a human-friendly group name <-> that wire blob so a user
// can write a directory group by name (and the provider resolves + encodes it),
// while a power user may still paste the raw base64 verbatim. See
// spike/DIRECTORY_SERVICE_GROUP_CRITERIA_SPIKE.md for the wire-probe results.
//
// Two questions are kept distinct (the "set A / set B" split):
//   - set A — "is this a directory-service group criterion at all?" Membership
//     triggers value encoding. Class-agnostic: the five canonical names below.
//   - set B — the per-object-class allowlist of names the corresponding Jamf API
//     surface actually accepts (computer 5 / mobile 4 / user 1). A name in A but
//     not in B's class is rejected at plan (fail-closed) with a clear message,
//     rather than shipping a literal group name the server would 409/400 on.
//
// The criterion names are the server-canonical TITLE-CASE forms enumerated by
// live probe (re-probed 2026-06-17 against platform-nmartin). Every surface —
// Platform device-groups AND the classic/Pro searches — is CASE-SENSITIVE on the
// whole string: the lowercase doc form ("Assigned user…") and an all-caps "MDM"
// are both rejected (400 INVALID_FIELD / 409 "Problem with criteria"). The MDM
// criterion's canonical token is "Mdm" (title-case), not "MDM" — an earlier
// spike probed it as "MDM", got rejected everywhere, and wrongly excluded it. It
// IS accepted (as "Mdm") on computer + mobile. The Jamf 11.29 doc table is NOT
// authoritative on casing.

// ObjectType is the inventory object class a criteria-bearing resource targets.
// It dispatches the per-class directory-service group allowlist (set B).
type ObjectType int

const (
	// ObjectTypeComputer covers device_group (computer) + advanced_computer_search.
	ObjectTypeComputer ObjectType = iota
	// ObjectTypeMobile covers device_group (mobile) + advanced_mobile_device_search.
	ObjectTypeMobile
	// ObjectTypeUser covers user_group + advanced_user_search.
	ObjectTypeUser
)

// Server-canonical directory-service group criterion names (exact, case-sensitive).
// Note "Mdm" is title-case, not "MDM" — the server rejects the all-caps form.
const (
	dsAssignedUser    = "Assigned User directory service group"
	dsUsername        = "Username directory service group"
	dsULLIComputer    = "User last logged in - Computer directory service group"
	dsULLISelfService = "User last logged in - Self Service directory service group"
	dsULLIMdm         = "User last logged in - Mdm directory service group"
)

// dsGroupNamesAll (set A) is every directory-service group criterion name across
// all object classes. Membership means "the value must be encoded".
var dsGroupNamesAll = map[string]bool{
	dsAssignedUser:    true,
	dsUsername:        true,
	dsULLIComputer:    true,
	dsULLISelfService: true,
	dsULLIMdm:         true,
}

// dsGroupAllowlist (set B) is the per-object-class set of accepted names, from
// the empirical matrix (re-probed 2026-06-17). Computer accepts all five; mobile
// accepts all but the Computer criterion (400 INVALID_FIELD); the user surfaces
// accept only Username (everything else 409 "Problem with criteria").
// advanced_computer_search ≡ device_group COMPUTER; advanced_mobile_device_search
// ≡ device_group MOBILE; advanced_user_search ≡ user_group.
var dsGroupAllowlist = map[ObjectType]map[string]bool{
	ObjectTypeComputer: {
		dsAssignedUser:    true,
		dsUsername:        true,
		dsULLIComputer:    true,
		dsULLISelfService: true,
		dsULLIMdm:         true,
	},
	ObjectTypeMobile: {
		dsAssignedUser:    true,
		dsUsername:        true,
		dsULLISelfService: true,
		dsULLIMdm:         true,
	},
	ObjectTypeUser: {
		dsUsername: true,
	},
}

// uuidPattern matches the canonical 8-4-4-4-12 hex UUID form. Case-insensitive:
// validation must not reject a valid-but-lowercase uuid (the wire is uppercase,
// but a pasted blob is preserved verbatim — see ResolveValue).
var uuidPattern = regexp.MustCompile(`^[0-9A-Fa-f]{8}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{12}$`)

// ErrNoUUID is returned when a directory group resolves but carries no uuid (the
// Okta default-mappings trap — Jamf rejects saving a smart group whose group has
// no uuid, so the provider must fail loudly rather than encode an empty uuid).
var ErrNoUUID = errors.New("directory group has no uuid mapping")

// dsGroupRef is the decoded criterion value: {"uuid":"…","serverId":"…"}. Field
// order matters: json.Marshal emits in declaration order, matching the UI's
// {"uuid","serverId"} byte layout so a resolved value round-trips byte-stable.
type dsGroupRef struct {
	UUID     string `json:"uuid"`
	ServerID string `json:"serverId"`
}

// IsDSGroupCriterion reports whether attributeName is a directory-service group
// criterion (set A) — i.e. its value must be encoded/decoded. Class-agnostic.
func IsDSGroupCriterion(attributeName string) bool {
	return dsGroupNamesAll[attributeName]
}

// isAllowedDSGroupCriterion reports whether attributeName is accepted on the
// given object class (set B).
func isAllowedDSGroupCriterion(ot ObjectType, attributeName string) bool {
	return dsGroupAllowlist[ot][attributeName]
}

// supportedDSGroupNames returns the accepted names for an object class, for use
// in the fail-closed error message.
func supportedDSGroupNames(ot ObjectType) []string {
	set := dsGroupAllowlist[ot]
	// Stable, canonical order matching the const declarations.
	ordered := []string{dsAssignedUser, dsUsername, dsULLIComputer, dsULLISelfService, dsULLIMdm}
	out := make([]string, 0, len(set))
	for _, n := range ordered {
		if set[n] {
			out = append(out, n)
		}
	}
	return out
}

// encodeDSGroupValue marshals a ref to its base64 wire form.
func encodeDSGroupValue(ref dsGroupRef) string {
	b, _ := json.Marshal(ref) // struct with string fields never fails to marshal
	return base64.StdEncoding.EncodeToString(b)
}

// EncodeDSGroupValue returns the base64 {uuid,serverId} wire value for a resolved
// directory-service group, matching exactly what ResolveValue produces from a
// name. Exposed as a deterministic, offline escape hatch (and for tests) so a
// caller holding a uuid + server id can build the raw value the API stores.
func EncodeDSGroupValue(uuid, serverID string) string {
	return encodeDSGroupValue(dsGroupRef{UUID: uuid, ServerID: serverID})
}

// ParseEncodedValue base64-decodes + JSON-parses value and validates the shape.
//
//   - ok=false, err=nil → "not an encoded value" → the caller treats value as a
//     group NAME (the §1a discriminator: a real group name does not base64-decode
//     to a {uuid,serverId} object).
//   - ok=true, err=nil → value is a valid pre-encoded blob → pass through verbatim.
//   - err!=nil → value LOOKED encoded (decoded to a JSON object carrying a uuid or
//     serverId key) but is malformed (empty/short field, bad uuid) → fail-closed.
//
// Offline (no API call).
func ParseEncodedValue(value string) (dsGroupRef, bool, error) {
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return dsGroupRef{}, false, nil // not base64 → a name
	}
	// Must decode to a JSON object; otherwise it is not our shape → a name.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(decoded, &raw); err != nil {
		return dsGroupRef{}, false, nil
	}
	_, hasUUID := raw["uuid"]
	_, hasServer := raw["serverId"]
	if !hasUUID && !hasServer {
		return dsGroupRef{}, false, nil // a JSON object, but not a DS-group ref → a name
	}
	// It looked encoded — validate strictly, error on any defect.
	var ref dsGroupRef
	if err := json.Unmarshal(decoded, &ref); err != nil {
		return dsGroupRef{}, false, fmt.Errorf("value decodes to base64 but is not a valid directory-service group reference: %w", err)
	}
	if ref.UUID == "" {
		return dsGroupRef{}, false, fmt.Errorf("encoded directory-service group value has an empty uuid (the directory lookup returned no uuid mapping — e.g. Okta with default mappings)")
	}
	if !uuidPattern.MatchString(ref.UUID) {
		return dsGroupRef{}, false, fmt.Errorf("encoded directory-service group value has a malformed uuid %q", ref.UUID)
	}
	if ref.ServerID == "" {
		return dsGroupRef{}, false, fmt.Errorf("encoded directory-service group value has an empty serverId")
	}
	return ref, true, nil
}

// ResolveValue is the write-path entry point for a directory-service group
// criterion value. If value is already a valid encoded blob it is returned
// verbatim (pass-through). Otherwise value is treated as a group NAME, resolved
// across all configured LDAP servers, and encoded.
//
// Hard errors (the config is wrong and must be fixed before apply succeeds):
// malformed base64 blob, group not found, ambiguous name (matches >1 server),
// or a resolved group with no uuid (ErrNoUUID).
func ResolveValue(ctx context.Context, resolver ldapgroups.Searcher, value string) (string, error) {
	_, ok, err := ParseEncodedValue(value)
	if err != nil {
		return "", err // looked encoded but malformed → fail-closed
	}
	if ok {
		return value, nil // already encoded → pass through verbatim
	}

	groups, err := ldapgroups.ResolveByName(ctx, resolver, value)
	if err != nil {
		return "", fmt.Errorf("resolving directory-service group %q: %w", value, err)
	}
	switch len(groups) {
	case 0:
		return "", fmt.Errorf("%w: %q (no directory group with this exact name on any configured LDAP server)", ldapgroups.ErrGroupNotFound, value)
	case 1:
		// ok
	default:
		return "", fmt.Errorf("%w: %q matches groups on %d LDAP servers — paste the base64 {uuid,serverId} value to disambiguate", ldapgroups.ErrAmbiguousGroup, value, len(groups))
	}

	g := groups[0]
	if g.UUID == "" {
		return "", fmt.Errorf("%w: %q (Jamf rejects saving a directory-service group criterion without a uuid)", ErrNoUUID, value)
	}
	return encodeDSGroupValue(dsGroupRef{UUID: g.UUID, ServerID: strconv.Itoa(g.LdapServerID)}), nil
}

// ResolveCriterionValue is the per-criterion write-path helper for consumers
// whose criterion model is NOT CriterionModel (device_group, user_group). It
// encapsulates the set-A trigger, the set-B fail-closed check, and value
// resolution:
//
//   - non-DS-group criterion → returns value unchanged, isDS=false, no error.
//   - DS-group criterion not accepted on this object class → isDS=true and a
//     fail-closed error listing the supported names.
//   - DS-group criterion accepted → resolves/encodes the value (isDS=true); a
//     resolution failure (not found / ambiguous / no uuid / malformed) is returned.
//
// Callers loop over their own model, set the returned wire value, and record an
// authored[wire]=originalValue entry when isDS so the read-back can restore the
// authored form (mirror RestoreAuthoredDSGroupCriteria / ReadValue).
func ResolveCriterionValue(ctx context.Context, resolver ldapgroups.Searcher, ot ObjectType, name, value string) (string, bool, error) {
	if !IsDSGroupCriterion(name) {
		return value, false, nil
	}
	if !isAllowedDSGroupCriterion(ot, name) {
		return value, true, fmt.Errorf("criterion %q is not supported on this resource; supported directory-service group criteria: %s",
			name, strings.Join(supportedDSGroupNames(ot), ", "))
	}
	wire, err := ResolveValue(ctx, resolver, value)
	if err != nil {
		return value, true, err
	}
	return wire, true, nil
}

// ReadValue maps a wire base64 value back to the state value, preserving the
// form the user authored. SOFT: it never propagates a resolution error — a prior
// name that no longer resolves (group deleted/renamed server-side, or a transient
// API blip) is surfaced as drift by returning the wire base64, so a single
// server-side change does not break every subsequent `terraform plan`.
//
//   - prior is empty (import / data source) → return wire (base64 is first-class).
//   - prior is itself an encoded blob → keep it if it equals wire, else wire (drift).
//   - prior is a NAME → re-resolve; if it re-encodes to wire, keep the name; else
//     return wire (the group changed → drift).
func ReadValue(ctx context.Context, resolver ldapgroups.Searcher, wire, prior string) string {
	if prior == "" {
		return wire
	}
	if _, ok, err := ParseEncodedValue(prior); ok && err == nil {
		if prior == wire {
			return prior
		}
		return wire // both base64, differ → drift
	}
	// prior is a name (or a malformed blob we treat as a name): re-resolve.
	encoded, err := ResolveValue(ctx, resolver, prior)
	if err != nil {
		return wire // soft: surface as drift, never error on read
	}
	if encoded == wire {
		return prior
	}
	return wire
}

// ---- CriterionModel layer (the shared CriterionModel consumers) -------------
//
// These three functions wire the primitives above into the []CriterionModel path
// shared by advanced_computer_search, advanced_user_search,
// advanced_mobile_device_search and user_group. device_group has its own model
// and wires the primitives directly.

// ResolveDSGroupCriteria resolves the value of every directory-service group
// criterion (set A) in models from a group NAME to the base64 wire form
// (pass-through when already encoded). Non-DS-group criteria are untouched.
//
// It returns a COPY of models with resolved values, plus an authored map keyed by
// the resolved wire value → the original authored value (name or base64). The map
// lets Create/Update restore the authored form after the read-back flatten without
// a second API call (RestoreAuthoredDSGroupCriteria), sidestepping any index
// realignment from the priority sort in BuildCriterionSlice.
//
// A name in set A but not accepted on this object class (set B) is rejected
// (fail-closed, spike §7.3); resolution failures (not found / ambiguous / no uuid
// / malformed blob) surface as errors. Call in Create/Update before Build*.
func ResolveDSGroupCriteria(ctx context.Context, resolver ldapgroups.Searcher, ot ObjectType, models []CriterionModel) ([]CriterionModel, map[string]string, diag.Diagnostics) {
	var diags diag.Diagnostics
	if len(models) == 0 {
		return models, nil, diags // preserve nil/empty; never flip nil -> []
	}
	authored := map[string]string{}
	out := make([]CriterionModel, len(models))
	copy(out, models)

	for i := range out {
		name := out[i].Name.ValueString()
		original := out[i].Value.ValueString()
		wire, isDS, err := ResolveCriterionValue(ctx, resolver, ot, name, original)
		if !isDS {
			continue
		}
		if err != nil {
			diags.AddError(
				"Invalid directory-service group criterion",
				fmt.Sprintf("Criterion %q: %s", name, err.Error()),
			)
			continue
		}
		out[i].Value = types.StringValue(wire)
		authored[wire] = original
	}
	return out, authored, diags
}

// RestoreAuthoredDSGroupCriteria rewrites each directory-service group
// criterion's value back to the form the user authored, using the map produced by
// ResolveDSGroupCriteria. Pure (no API call, no error): the wire echoes the value
// we just wrote byte-stable, so this is a lookup-and-restore. Call in Create/Update
// after the read-back flatten. Criteria absent from the map (or not DS-group) are
// left as flattened.
func RestoreAuthoredDSGroupCriteria(models []CriterionModel, authored map[string]string) []CriterionModel {
	if len(authored) == 0 {
		return models
	}
	out := make([]CriterionModel, len(models))
	copy(out, models)
	for i := range out {
		if !IsDSGroupCriterion(out[i].Name.ValueString()) {
			continue
		}
		if v, ok := authored[out[i].Value.ValueString()]; ok {
			out[i].Value = types.StringValue(v)
		}
	}
	return out
}

// DSGroupValuesEquivalent reports whether two directory-service group criterion
// values refer to the SAME group — i.e. each resolves to the same wire base64
// ({uuid,serverId}), where a raw base64 value passes through and a name is
// resolved. Used at plan time to suppress a no-op base64<->name swap.
//
// SOFT and class-agnostic: identical strings short-circuit true with no API call;
// any resolution failure (not found / ambiguous / transient) returns false so a
// transient LDAP blip can never block `terraform plan` — it just shows the diff.
func DSGroupValuesEquivalent(ctx context.Context, resolver ldapgroups.Searcher, a, b string) bool {
	if a == b {
		return true
	}
	wa, err := ResolveValue(ctx, resolver, a)
	if err != nil {
		return false
	}
	wb, err := ResolveValue(ctx, resolver, b)
	if err != nil {
		return false
	}
	if wa == wb {
		return true
	}
	// Fall back to a semantic compare so two encodings of the same group — e.g. a
	// pasted blob with a lowercase uuid or different JSON key order — still match.
	ra, oka, _ := ParseEncodedValue(wa)
	rb, okb, _ := ParseEncodedValue(wb)
	return oka && okb && strings.EqualFold(ra.UUID, rb.UUID) && ra.ServerID == rb.ServerID
}

// SuppressEquivalentDSGroupValues returns a copy of planned where each
// directory-service group criterion whose value is semantically equivalent to the
// aligned prior value is reset to the prior value — suppressing a no-op
// base64<->name diff so `terraform plan` shows nothing to change. planned and
// prior are aligned by index, guarded by a name match (mirrors
// ReadbackDSGroupCriteria); a reorder simply degrades to "no suppression" (the
// diff shows, which is safe). Set-B validation is deliberately NOT done here — it
// stays at apply (ResolveCriterionValue) so a transient resolve failure cannot
// turn into a hard plan error. Call from a resource's ModifyPlan.
func SuppressEquivalentDSGroupValues(ctx context.Context, resolver ldapgroups.Searcher, planned, prior []CriterionModel) []CriterionModel {
	if len(planned) == 0 {
		return planned // preserve nil/empty
	}
	out := make([]CriterionModel, len(planned))
	copy(out, planned)
	for i := range out {
		name := out[i].Name.ValueString()
		if !IsDSGroupCriterion(name) {
			continue
		}
		if i >= len(prior) || prior[i].Name.ValueString() != name {
			continue
		}
		if DSGroupValuesEquivalent(ctx, resolver, out[i].Value.ValueString(), prior[i].Value.ValueString()) {
			out[i] = reconcileCriterionToPrior(out[i], prior[i])
		}
	}
	return out
}

// reconcileCriterionToPrior collapses a criterion whose value is a CONFIRMED
// representation swap (base64<->name for the same group) back to the prior state
// element: the value adopts the stored form, and any sibling field the plan left
// Unknown is filled from prior. The Unknown case is load-bearing — `priority` is
// Optional+Computed with no default, so core flips it to Unknown the moment the
// criteria list "changes" (the value swap); without this it would stay Unknown
// (!= the stored value) and defeat the no-op, leaving a phantom criteria diff.
// Known fields the user actually changed are preserved, so a real edit made
// alongside the swap still surfaces a diff.
func reconcileCriterionToPrior(plan, prior CriterionModel) CriterionModel {
	plan.Value = prior.Value
	if plan.Priority.IsUnknown() {
		plan.Priority = prior.Priority
	}
	if plan.Name.IsUnknown() {
		plan.Name = prior.Name
	}
	if plan.SearchType.IsUnknown() {
		plan.SearchType = prior.SearchType
	}
	if plan.AndOr.IsUnknown() {
		plan.AndOr = prior.AndOr
	}
	if plan.HasOpeningParenthesis.IsUnknown() {
		plan.HasOpeningParenthesis = prior.HasOpeningParenthesis
	}
	if plan.HasClosingParenthesis.IsUnknown() {
		plan.HasClosingParenthesis = prior.HasClosingParenthesis
	}
	return plan
}

// CriteriaModelsEqual reports whether two criterion slices are identical across
// every modelled field. ModifyPlan suppression uses it to confirm a representation
// swap left no real criteria change before restoring Computed siblings (e.g.
// site_name) that core flips to unknown on any planned update.
func CriteriaModelsEqual(a, b []CriterionModel) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !a[i].Priority.Equal(b[i].Priority) ||
			!a[i].Name.Equal(b[i].Name) ||
			!a[i].SearchType.Equal(b[i].SearchType) ||
			!a[i].Value.Equal(b[i].Value) ||
			!a[i].AndOr.Equal(b[i].AndOr) ||
			!a[i].HasOpeningParenthesis.Equal(b[i].HasOpeningParenthesis) ||
			!a[i].HasClosingParenthesis.Equal(b[i].HasClosingParenthesis) {
			return false
		}
	}
	return true
}

// ReadbackDSGroupCriteria maps each directory-service group criterion's wire
// value back to the authored form on a pure Read/refresh, using prior state as the
// form source (SOFT — never errors; see ReadValue). wireModels are the freshly
// flattened (base64) criteria; priorModels are the existing state criteria. Both
// are wire-ordered, so they zip by index; the index is guarded by a name match.
// When there is no aligned prior (import / data source / drift), the wire base64
// is kept. Call in Read after the flatten.
func ReadbackDSGroupCriteria(ctx context.Context, resolver ldapgroups.Searcher, ot ObjectType, wireModels, priorModels []CriterionModel) []CriterionModel {
	if len(wireModels) == 0 {
		return wireModels // preserve nil/empty; never flip nil -> []
	}
	out := make([]CriterionModel, len(wireModels))
	copy(out, wireModels)
	for i := range out {
		name := out[i].Name.ValueString()
		if !IsDSGroupCriterion(name) {
			continue
		}
		prior := ""
		if i < len(priorModels) && priorModels[i].Name.ValueString() == name {
			prior = priorModels[i].Value.ValueString()
		}
		out[i].Value = types.StringValue(ReadValue(ctx, resolver, out[i].Value.ValueString(), prior))
	}
	return out
}
