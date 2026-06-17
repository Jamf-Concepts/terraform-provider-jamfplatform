// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package device_group

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/criteria"
)

// Jamf-group "member of" criteria on a Platform device group reference a Jamf
// group by NAME as their attributeValue. Jamf Pro 11.29 echoes that value back as
// the group's numeric id on read for the COMPUTER device type (criterion
// "Computer Group"); MOBILE round-trips the name unchanged (wire-probed — see
// spike/JAMF_GROUP_MEMBER_OF_CRITERIA_SPIKE.md). These thin per-criterion loops
// wire the shared value-level helpers (criteria.ReadGroupValue / GroupValuesEquivalent)
// into device_group's own criterion model (DeviceGroupCriteriaModel, value field
// AttributeValue), exactly as dsgroup.go does for the directory-service family.
// The two families have disjoint criterion names, so they never touch the same
// element. The mapping is version-agnostic: when wire==config it short-circuits
// with no lookup, so a pre-11.29 server (name) and the clean MOBILE path are both
// no-ops.

// resolveGroupRefWireIDs returns a COPY of in with each Jamf-group criterion's
// value rewritten from the group NAME to its numeric ID for the wire, plus an
// id->authoredValue map for the post-flatten restore. device_group's UPDATE (PATCH)
// endpoint REQUIRES the id — wire-probed: PATCH with a name 400s ("Computer/Mobile
// Group depends on Group (<name>) which does not exist"), while the id succeeds on
// both POST and PATCH. So unlike user_group (whose classic PUT resolves names), the
// device_group wire value is always the id. Applies to both device types (mobile
// regresses identically once updated). Best-effort/SOFT: a value that does not
// resolve (already an id, or an unknown name) is passed through unchanged — an
// unknown name then surfaces the server's clear "does not exist" error rather than
// being masked.
func resolveGroupRefWireIDs(ctx context.Context, resolver criteria.GroupResolver, ot criteria.ObjectType, in []DeviceGroupCriteriaModel) ([]DeviceGroupCriteriaModel, map[string]string) {
	if resolver == nil || len(in) == 0 {
		return in, nil
	}
	authored := map[string]string{}
	out := make([]DeviceGroupCriteriaModel, len(in))
	copy(out, in)
	for i := range out {
		name := out[i].AttributeName.ValueString()
		if !criteria.IsJamfGroupCriterion(ot, name, out[i].Operator.ValueString()) {
			continue
		}
		v := out[i].AttributeValue.ValueString()
		if id, err := resolver.IDByName(ctx, ot, v); err == nil && id != "" {
			out[i].AttributeValue = types.StringValue(id) // send the id on the wire
			authored[id] = v                              // restore the authored name on read-back
		} else {
			authored[v] = v // already an id (or unresolved name) -> send as-is
		}
	}
	return out, authored
}

// restoreAuthoredGroupRefCriteria rewrites each Jamf-group criterion's flattened
// wire value back to the authored value using the map from
// resolveAuthoredGroupRefMap. The wire id is by definition the id of the group we
// just wrote by name, so the restore is unconditional (no resolver call, no
// post-apply inconsistency risk). Call in Create/Update after the flatten.
func restoreAuthoredGroupRefCriteria(in []DeviceGroupCriteriaModel, authored map[string]string, ot criteria.ObjectType) []DeviceGroupCriteriaModel {
	if len(authored) == 0 || len(in) == 0 {
		return in
	}
	out := make([]DeviceGroupCriteriaModel, len(in))
	copy(out, in)
	for i := range out {
		if !criteria.IsJamfGroupCriterion(ot, out[i].AttributeName.ValueString(), out[i].Operator.ValueString()) {
			continue
		}
		if v, ok := authored[out[i].AttributeValue.ValueString()]; ok {
			out[i].AttributeValue = types.StringValue(v)
		}
	}
	return out
}

// readbackGroupRefCriteria maps each Jamf-group criterion's wire value back to the
// authored group name on a pure Read, using prior state as the form source (SOFT —
// never errors; backward-compatible — wire==prior short-circuits with no lookup).
// Call in Read after the flatten.
func readbackGroupRefCriteria(ctx context.Context, resolver criteria.GroupResolver, ot criteria.ObjectType, in, prior []DeviceGroupCriteriaModel) []DeviceGroupCriteriaModel {
	if len(in) == 0 || resolver == nil {
		return in // preserve nil/empty; never flip nil -> []
	}
	out := make([]DeviceGroupCriteriaModel, len(in))
	copy(out, in)
	for i := range out {
		name := out[i].AttributeName.ValueString()
		if !criteria.IsJamfGroupCriterion(ot, name, out[i].Operator.ValueString()) {
			continue
		}
		p := ""
		if i < len(prior) && prior[i].AttributeName.ValueString() == name {
			p = prior[i].AttributeValue.ValueString()
		}
		out[i].AttributeValue = types.StringValue(criteria.ReadGroupValue(ctx, resolver, ot, out[i].AttributeValue.ValueString(), p))
	}
	return out
}

// suppressEquivalentGroupRefCriteria resets a planned Jamf-group criterion value to
// the prior state value when both reference the same group (a name<->id swap),
// suppressing a no-op diff. Aligned by index + name match; SOFT. Call from
// ModifyPlan.
func suppressEquivalentGroupRefCriteria(ctx context.Context, resolver criteria.GroupResolver, ot criteria.ObjectType, planned, prior []DeviceGroupCriteriaModel) []DeviceGroupCriteriaModel {
	if len(planned) == 0 || resolver == nil {
		return planned // preserve nil/empty
	}
	out := make([]DeviceGroupCriteriaModel, len(planned))
	copy(out, planned)
	for i := range out {
		name := out[i].AttributeName.ValueString()
		if !criteria.IsJamfGroupCriterion(ot, name, out[i].Operator.ValueString()) {
			continue
		}
		if i >= len(prior) || prior[i].AttributeName.ValueString() != name {
			continue
		}
		if criteria.GroupValuesEquivalent(ctx, resolver, ot, out[i].AttributeValue.ValueString(), prior[i].AttributeValue.ValueString()) {
			out[i].AttributeValue = prior[i].AttributeValue
		}
	}
	return out
}
