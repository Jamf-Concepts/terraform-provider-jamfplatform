// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package device_group

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/criteria"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/ldapgroups"
)

// Directory-service group criteria carry a base64 {uuid,serverId} blob as their
// value (see internal/common/criteria/dsgroup.go). device_group has its own
// criterion model (DeviceGroupCriteriaModel, value field AttributeValue) rather
// than the shared criteria.CriterionModel, so these thin per-criterion loops
// wire the shared primitives directly. The operator is upper-cased to MEMBER OF
// by expandDeviceGroupCriteria already (the Platform surface wants UPPERCASE),
// so only the value is transformed here.

// dsObjectType maps the resource's device_type to the criteria object class that
// dispatches the per-class directory-service group allowlist.
func dsObjectType(deviceType string) criteria.ObjectType {
	if strings.EqualFold(deviceType, "mobile") {
		return criteria.ObjectTypeMobile
	}
	return criteria.ObjectTypeComputer
}

// resolveDSGroupCriteria resolves each directory-service group criterion's value
// from a group NAME to the base64 wire form (pass-through when already encoded),
// fail-closing on a name not accepted for this object class. Returns the resolved
// models plus an authored map (wire→original) for restoring the authored form
// after the post-write flatten. Call in Create/Update before expand.
func resolveDSGroupCriteria(ctx context.Context, resolver ldapgroups.Searcher, ot criteria.ObjectType, in []DeviceGroupCriteriaModel) ([]DeviceGroupCriteriaModel, map[string]string, diag.Diagnostics) {
	var diags diag.Diagnostics
	if len(in) == 0 {
		return in, nil, diags // preserve nil/empty
	}
	authored := map[string]string{}
	out := make([]DeviceGroupCriteriaModel, len(in))
	copy(out, in)
	for i := range out {
		name := out[i].AttributeName.ValueString()
		original := out[i].AttributeValue.ValueString()
		wire, isDS, err := criteria.ResolveCriterionValue(ctx, resolver, ot, name, original)
		if !isDS {
			continue
		}
		if err != nil {
			diags.AddError("Invalid directory-service group criterion", fmt.Sprintf("Criterion %q: %s", name, err.Error()))
			continue
		}
		out[i].AttributeValue = types.StringValue(wire)
		authored[wire] = original
	}
	return out, authored, diags
}

// restoreAuthoredDSGroupCriteria rewrites each directory-service group criterion
// value back to the authored form using the map from resolveDSGroupCriteria.
// Pure (the wire echoes the value we wrote byte-stable). Call after the flatten.
func restoreAuthoredDSGroupCriteria(in []DeviceGroupCriteriaModel, authored map[string]string) []DeviceGroupCriteriaModel {
	if len(authored) == 0 {
		return in
	}
	out := make([]DeviceGroupCriteriaModel, len(in))
	copy(out, in)
	for i := range out {
		if !criteria.IsDSGroupCriterion(out[i].AttributeName.ValueString()) {
			continue
		}
		if v, ok := authored[out[i].AttributeValue.ValueString()]; ok {
			out[i].AttributeValue = types.StringValue(v)
		}
	}
	return out
}

// readbackDSGroupCriteria maps each directory-service group criterion's wire
// value back to the authored form on a pure Read, using prior state as the form
// source (SOFT — never errors). Call in Read after the flatten.
func readbackDSGroupCriteria(ctx context.Context, resolver ldapgroups.Searcher, in, prior []DeviceGroupCriteriaModel) []DeviceGroupCriteriaModel {
	if len(in) == 0 {
		return in // preserve nil/empty; never flip nil -> []
	}
	out := make([]DeviceGroupCriteriaModel, len(in))
	copy(out, in)
	for i := range out {
		name := out[i].AttributeName.ValueString()
		if !criteria.IsDSGroupCriterion(name) {
			continue
		}
		p := ""
		if i < len(prior) && prior[i].AttributeName.ValueString() == name {
			p = prior[i].AttributeValue.ValueString()
		}
		out[i].AttributeValue = types.StringValue(criteria.ReadValue(ctx, resolver, out[i].AttributeValue.ValueString(), p))
	}
	return out
}

// suppressEquivalentDSGroupCriteria resets a planned directory-service group
// criterion value to the prior state value when the two refer to the same group
// (a base64<->name swap), suppressing a no-op diff. Aligned by index + name
// match; SOFT. Call from ModifyPlan.
func suppressEquivalentDSGroupCriteria(ctx context.Context, resolver ldapgroups.Searcher, planned, prior []DeviceGroupCriteriaModel) []DeviceGroupCriteriaModel {
	if len(planned) == 0 {
		return planned // preserve nil/empty
	}
	out := make([]DeviceGroupCriteriaModel, len(planned))
	copy(out, planned)
	for i := range out {
		name := out[i].AttributeName.ValueString()
		if !criteria.IsDSGroupCriterion(name) {
			continue
		}
		if i >= len(prior) || prior[i].AttributeName.ValueString() != name {
			continue
		}
		if criteria.DSGroupValuesEquivalent(ctx, resolver, out[i].AttributeValue.ValueString(), prior[i].AttributeValue.ValueString()) {
			out[i].AttributeValue = prior[i].AttributeValue
		}
	}
	return out
}
