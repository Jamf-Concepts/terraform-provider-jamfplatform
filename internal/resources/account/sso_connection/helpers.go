// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package sso_connection

import (
	"context"
	"encoding/json"
	"sort"
	"strings"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform"
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/account"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// appendWriteDiagnostics turns a create or update failure into the most specific
// diagnostic the error body supports, and reports whether it recognised one.
//
// The first case is the one every write currently takes. An internal failure from
// this operation is not a hint to change the configuration: it was proven to
// happen before Jamf inspects the request at all, since the same failure comes
// back for an update aimed at an identifier that does not exist while a read and
// a withdrawal of that identifier both resolve it and answer not-found. So the
// diagnostic says the fault is Jamf's, says what still works, and names the
// identifier to quote, rather than sending an operator round a configuration they
// cannot fix.
//
// The remaining cases are the two Jamf does attribute. An unrecognised
// vocabulary value is refused with the offending value named in the message but
// no field named at all, so the message is passed through verbatim — it is the
// only thing that localises the problem. A missing top-level field is one of the
// three Jamf names, and reaches an operator only through a configuration that
// resolved to nothing.
func appendWriteDiagnostics(diags *diag.Diagnostics, action string, err error) bool {
	apiErr := jamfplatform.AsAPIError(err)
	if apiErr == nil {
		return false
	}
	matched := false
	for _, detail := range apiErr.Details() {
		switch detail.Code {
		case codeUpstreamError:
			diags.AddError(
				"Jamf Account cannot "+action+" an SSO connection",
				"Jamf refused to "+action+" the connection with an internal failure that carries no detail. This "+
					"is a known fault on Jamf's side, not a problem with this configuration: every attempt to "+
					"create or change a connection is refused the same way, in every region, and the refusal "+
					"happens before the request is examined — an update aimed at an identifier that does not "+
					"exist is refused identically, while reading or destroying that same identifier reports it "+
					"missing as it should.\n\n"+
					"Reading, listing and destroying connections all work. Until Jamf fixes this, make and edit "+
					"connections in the Jamf Account console and use this provider to read them. If you raise it "+
					"with Jamf Support, quote the trace identifier: "+traceIDOrUnknown(apiErr)+". Reported by "+
					"Jamf Account: "+detail.Description,
			)
		case codeBadRequest:
			diags.AddError(
				"Jamf Account refused a value on the connection",
				"Jamf refused one of the values on this connection and does not say which attribute it was on — "+
					"it names only the value. Reported by Jamf Account: "+detail.Description,
			)
		case codeFieldValidation:
			diags.AddError(
				"A required part of the connection did not reach Jamf Account",
				"Jamf reports a required part of the connection as missing or empty. This usually means a "+
					"variable or a reference resolved to nothing — `domains` in particular has to hold at least "+
					"one verified domain name. Reported by Jamf Account: "+detail.Description,
			)
		default:
			continue
		}
		matched = true
	}
	return matched
}

// traceIDOrUnknown renders the trace identifier from a refusal, or says there was
// none, so the diagnostic can always tell an operator what to quote.
func traceIDOrUnknown(apiErr *jamfplatform.APIResponseError) string {
	if id := apiErr.TraceID; id != "" {
		return id
	}
	return "none was returned"
}

// consentFlowExplanation is the one fact both consent-flow diagnostics rest on,
// stated once so the two cannot drift apart.
const consentFlowExplanation = "A connection set up with Microsoft's admin-consent flow has no client " +
	"registration of its own — Jamf holds the consent — and nothing this provider can send expresses that, so " +
	"any change Terraform tried to apply would be refused or would silently replace the consent with a client " +
	"that does not exist."

// appendConsentFlowDiagnostics refuses a connection built through Microsoft's
// admin-consent flow, and reports whether it did.
//
// Such a connection reads back cleanly and looks ordinary in every attribute an
// operator would think to check, which is exactly why it needs refusing where it
// is first seen rather than where it first fails. Letting one into state buys an
// entry that can never be applied: it has no client identifier, the settings
// Jamf accepts have no way to declare the consent, and the only escape is
// `terraform state rm` by hand.
//
// The refusal lands on Read, which covers both the read that follows an import
// and an ordinary refresh, and on Update, so a refresh Terraform skipped cannot
// let one through. Destroy is deliberately allowed: withdrawal works, and
// refusing it would trap an operator who already holds one.
func appendConsentFlowDiagnostics(diags *diag.Diagnostics, c *account.Connection) bool {
	if c == nil || !c.ConsentFlow {
		return false
	}
	diags.AddAttributeError(
		path.Root("consent_flow"),
		"Connection uses Microsoft admin consent and cannot be managed",
		"The connection \""+c.Name+"\" was set up with Microsoft's admin-consent flow, so "+
			"jamfplatform_account_sso_connection cannot manage it. "+consentFlowExplanation+" Read it with the "+
			"`jamfplatform_account_sso_connection` data source instead, which reports such a connection without "+
			"claiming to own it, and make any change in the Jamf Account console. If Terraform already holds it, "+
			"run `terraform state rm` on the address to drop it.",
	)
	return true
}

// appendConsentFlowUpdateDiagnostics refuses to change a connection built with
// Microsoft admin consent.
//
// Keyed on the value already in state rather than on a refusal from Jamf, which
// makes it deterministic and needs no guess about how such a write would be
// answered. That guess would be a particularly bad one at the moment: every write
// is refused identically whatever the reason, so a wire-keyed check could not
// tell this apart from the fault affecting every connection. Read refuses such a
// connection before it can reach state at all, so this covers the state an
// earlier provider version could have written, and it runs before the request so
// no doomed write is ever issued.
func appendConsentFlowUpdateDiagnostics(diags *diag.Diagnostics, name string) {
	diags.AddError(
		"Connection using Microsoft admin consent cannot be changed",
		"Terraform holds \""+name+"\" as a managed connection, but Jamf Account reports it as using Microsoft's "+
			"admin-consent flow. "+consentFlowExplanation+" Nothing Terraform can send would change it, so no "+
			"request was made. Change it in the Jamf Account console, run `terraform state rm` on this address "+
			"to drop it from state, and read it with the `jamfplatform_account_sso_connection` data source if "+
			"you still need its values.",
	)
}

// appendGhostConnectionDiagnostics reports a connection the organization's
// collection lists but which cannot be read on its own identifier.
//
// This is a disagreement inside Jamf between its own store and whatever the
// single-connection read consults, observed on one connection in the probe
// organization. The important half is what this must *not* do: treat the
// not-found as a withdrawal and drop the resource. It exists — the collection is
// Jamf's own record of it — so removing it from state would plan a fresh create
// of a connection that is already there, and the create would collide or, worse,
// leave two.
func appendGhostConnectionDiagnostics(diags *diag.Diagnostics, id, name string) {
	diags.AddError(
		"Jamf Account cannot read this SSO connection on its own identifier",
		"The connection \""+name+"\" (identifier "+id+") is listed among your organization's connections, but "+
			"reading it on that identifier reports it missing. That is a disagreement inside Jamf between its "+
			"own list and the record behind a single connection, not a sign the connection is gone — so "+
			"Terraform has deliberately left it in state rather than planning to create it again, which would "+
			"risk a duplicate.\n\n"+
			"Nothing in this configuration can fix it. Raise it with Jamf Support, quoting the identifier above. "+
			"Meanwhile `terraform state rm` on this address will stop Terraform trying to refresh it.",
	)
}

// findSummary returns the collection entry for one identifier, or nil.
func findSummary(summaries []account.ConnectionSummary, id string) *account.ConnectionSummary {
	for i := range summaries {
		if summaries[i].ID == id {
			return &summaries[i]
		}
	}
	return nil
}

// findSummariesByName returns every collection entry whose stored name matches.
//
// It returns all of them rather than the first because the stored name is not a
// unique key: Jamf may hold a uniquified form of whatever name was configured, so
// two connections asked for the same name can both answer to a name lookup. A
// caller reporting an ambiguity is more useful than one silently picking.
func findSummariesByName(summaries []account.ConnectionSummary, name string) []account.ConnectionSummary {
	var out []account.ConnectionSummary
	for i := range summaries {
		if summaries[i].Name == name {
			out = append(out, summaries[i])
		}
	}
	return out
}

// summaryNames renders the stored names of a collection for a diagnostic, sorted
// so the list is stable.
func summaryNames(summaries []account.ConnectionSummary) string {
	names := make([]string, 0, len(summaries))
	for _, s := range summaries {
		names = append(names, "\""+s.Name+"\" ("+s.ID+")")
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

// splitFilterGroups turns Jamf's comma-separated group list into its members.
//
// An empty string yields an empty slice rather than one empty member, because an
// operator with no groups is a real configuration and has to round-trip as an
// empty set. Members are trimmed, since nothing guarantees Jamf stores them
// without spaces around the separators.
func splitFilterGroups(csv string) []string {
	if strings.TrimSpace(csv) == "" {
		return []string{}
	}
	parts := strings.Split(csv, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

// joinFilterGroups renders group names as Jamf's comma-separated list, sorted so
// the same set always produces the same string. The attribute is a set, whose
// members reach the provider in an arbitrary order, so sorting is what keeps a
// write deterministic.
func joinFilterGroups(groups []string) string {
	sorted := make([]string, len(groups))
	copy(sorted, groups)
	sort.Strings(sorted)
	return strings.Join(sorted, ",")
}

// decodeJSONObject parses a JSON object string.
func decodeJSONObject(raw string) (map[string]any, error) {
	var decoded map[string]any
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return nil, err
	}
	return decoded, nil
}

// equivalentJSON reports whether two strings describe the same JSON.
//
// A value that will not parse is compared byte-wise, so a malformed document is
// still stable in state rather than looking different from itself. The
// attribute's validator is what reports it, against the right attribute.
func equivalentJSON(left, right string) bool {
	var decodedLeft, decodedRight any
	if json.Unmarshal([]byte(left), &decodedLeft) != nil || json.Unmarshal([]byte(right), &decodedRight) != nil {
		return left == right
	}
	return sameJSON(decodedLeft, decodedRight)
}

// sameJSON compares two decoded JSON values structurally, treating numbers by
// value so 1 and 1.0 agree.
func sameJSON(a, b any) bool {
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

// preferEquivalentJSON keeps the value Terraform planned when Jamf's copy says
// the same thing in different bytes.
//
// Jamf re-serialises the claim mapping on the way out, and `jsonencode` emits
// keys sorted and unindented, so the two forms differ constantly while meaning
// the same. Reconciling here rather than through a semantic-equality type is
// deliberate: semantic equality is never consulted while planning, so it cannot
// settle a difference between a configuration and prior state, whereas this runs
// on every path that writes state and settles all of them the same way.
func preferEquivalentJSON(planned types.String, fromWire *string) types.String {
	incoming := stringOrNull(fromWire)
	if planned.IsNull() || planned.IsUnknown() || incoming.IsNull() {
		return incoming
	}
	if equivalentJSON(planned.ValueString(), incoming.ValueString()) {
		return planned
	}
	return incoming
}

// stringOrNull renders an optional string, treating an empty one as absent —
// which is how Jamf reports a field it holds no value for.
func stringOrNull(v *string) types.String {
	if v == nil || *v == "" {
		return types.StringNull()
	}
	return types.StringValue(*v)
}

// boolOrNull renders an optional boolean.
func boolOrNull(v *bool) types.Bool {
	if v == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*v)
}

// int64OrNull renders an optional whole number.
func int64OrNull(v *int) types.Int64 {
	if v == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(*v))
}

// omitLoginHintFromWire inverts Jamf's login-hint flag into the console's
// question.
//
// Jamf records whether the hint is *forwarded*; the console asks whether it is
// *omitted*. See the package doc for why the console's sense is the one this
// provider exposes. The inverse is omitLoginHintToWire, and both directions are
// pinned by unit tests because a pair that inverts twice is indistinguishable
// from one that never inverts.
func omitLoginHintFromWire(aliasToIdp bool) types.Bool {
	return types.BoolValue(!aliasToIdp)
}

// omitLoginHintToWire inverts the console's question back into Jamf's flag.
//
// An unset value yields false, meaning the hint is forwarded: that is Jamf's own
// default and the console's unticked box, so an operator who says nothing gets
// the smoother sign-in.
func omitLoginHintToWire(omit types.Bool) bool {
	if omit.IsNull() || omit.IsUnknown() {
		return true
	}
	return !omit.ValueBool()
}

// setToStrings reads a string set into a slice, treating an absent set as empty.
func setToStrings(ctx context.Context, set types.Set) ([]string, diag.Diagnostics) {
	if set.IsNull() || set.IsUnknown() {
		return []string{}, nil
	}
	var out []string
	diags := set.ElementsAs(ctx, &out, false)
	if out == nil {
		out = []string{}
	}
	return out, diags
}

// stringsToSet builds a string set, always known so a Computed attribute resolves
// at apply.
func stringsToSet(values []string) (types.Set, diag.Diagnostics) {
	elements := make([]attr.Value, 0, len(values))
	for _, v := range values {
		elements = append(elements, types.StringValue(v))
	}
	return types.SetValue(types.StringType, elements)
}
