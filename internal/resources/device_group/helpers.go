// Copyright 2025 Jamf Software LLC.

package device_group

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// refreshDeviceGroupState retrieves the device group from the API and updates the Terraform state.
func (r *DeviceGroupResource) refreshDeviceGroupState(ctx context.Context, id string, model *DeviceGroupResourceModel, manageMembers bool, manageDescription bool, diags *diag.Diagnostics) bool {
	grp, err := r.client.GetDeviceGroupByIDV1(ctx, id)
	if err != nil {
		diags.AddError("Error reading device group", err.Error())
		return false
	}

	var members []string
	if strings.EqualFold(grp.GroupType, "STATIC") && manageMembers {
		var err error
		members, err = r.client.GetDeviceGroupMembersV1(ctx, grp.ID)
		if err != nil {
			diags.AddError("Error reading device group members", err.Error())
			return false
		}
	}

	diags.Append(assignDeviceGroupModel(ctx, model, grp, members, manageMembers, manageDescription)...)
	return !diags.HasError()
}

// waitForDeviceGroupAvailability polls the API until the newly created device group is available.
func (r *DeviceGroupResource) waitForDeviceGroupAvailability(ctx context.Context, id string) error {
	var lastErr error
	for attempt := 1; attempt <= deviceGroupCreateMaxAttempts; attempt++ {
		_, err := r.client.GetDeviceGroupByIDV1(ctx, id)
		if err == nil {
			return nil
		}
		if !isNotFoundError(err) {
			return err
		}

		lastErr = err
		if attempt == deviceGroupCreateMaxAttempts {
			break
		}

		tflog.Debug(ctx, "device group not yet available, retrying", map[string]interface{}{
			"id":      id,
			"attempt": attempt,
		})

		timer := time.NewTimer(deviceGroupCreateRetryDelay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("context cancelled while waiting for device group %s: %w", id, ctx.Err())
		case <-timer.C:
		}
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("device group %s not ready", id)
	}

	return fmt.Errorf("device group %s not yet available after %d attempts: %w", id, deviceGroupCreateMaxAttempts, lastErr)
}

// assignDeviceGroupModel maps API representation to Terraform model, respecting managed fields.
func assignDeviceGroupModel(ctx context.Context, model *DeviceGroupResourceModel, grp *client.DeviceGroupReadRepresentationV1, members []string, manageMembers bool, manageDescription bool) diag.Diagnostics {
	var diags diag.Diagnostics

	prevDescription := model.Description
	prevCriteria := model.Criteria
	model.ID = types.StringValue(grp.ID)
	if manageDescription {
		model.Description = reconcileOptionalString(grp.Description, prevDescription)
	} else {
		model.Description = types.StringNull()
	}
	if grp.DeviceType == "" {
		model.DeviceType = types.StringNull()
	} else {
		model.DeviceType = types.StringValue(strings.ToLower(grp.DeviceType))
	}

	if grp.GroupType == "" {
		model.GroupType = types.StringNull()
	} else {
		model.GroupType = types.StringValue(strings.ToLower(grp.GroupType))
	}
	model.MemberCount = types.Int64Value(int64(grp.MemberCount))
	model.Criteria = flattenDeviceGroupCriteria(grp.Criteria, prevCriteria)

	if strings.EqualFold(grp.GroupType, "STATIC") {
		if manageMembers {
			memberValues := members
			if memberValues == nil {
				memberValues = []string{}
			}
			set, setDiags := types.SetValueFrom(ctx, types.StringType, memberValues)
			diags.Append(setDiags...)
			if !diags.HasError() {
				model.Members = set
			}
		} else {
			model.Members = types.SetNull(types.StringType)
		}
	} else {
		model.Members = types.SetNull(types.StringType)
	}

	return diags
}

// validateDeviceGroupPlan checks that the plan is valid based on group type.
func validateDeviceGroupPlan(plan *DeviceGroupResourceModel) error {
	groupType := strings.ToLower(plan.GroupType.ValueString())

	switch groupType {
	case "smart":
		if len(plan.Criteria) == 0 {
			return fmt.Errorf("criteria must be supplied for smart groups")
		}
		if !plan.Members.IsNull() && !plan.Members.IsUnknown() {
			return fmt.Errorf("members cannot be set for smart groups")
		}
	case "static":
		if len(plan.Criteria) > 0 {
			return fmt.Errorf("criteria cannot be set for static groups")
		}
		if !plan.Members.IsNull() && plan.Members.IsUnknown() {
			return fmt.Errorf("members cannot be unknown when provided")
		}
	default:
		if groupType == "" {
			return fmt.Errorf("group_type must be provided")
		}
		return fmt.Errorf("unsupported group_type %q", groupType)
	}

	return nil
}

// expandDeviceGroupCriteria converts Terraform models into API representations.
func expandDeviceGroupCriteria(criteria []DeviceGroupCriteriaModel) []client.DeviceGroupCriteriaRepresentationV1 {
	if len(criteria) == 0 {
		return nil
	}
	result := make([]client.DeviceGroupCriteriaRepresentationV1, 0, len(criteria))
	for idx, c := range criteria {
		if c.AttributeName.IsNull() || c.AttributeName.IsUnknown() {
			continue
		}

		operator := ""
		if isConfiguredValue(c.Operator) {
			operator = strings.ToUpper(c.Operator.ValueString())
		}

		attributeValue := ""
		if isConfiguredValue(c.AttributeValue) {
			attributeValue = c.AttributeValue.ValueString()
		}

		joinType := ""
		if isConfiguredValue(c.JoinType) {
			joinType = strings.ToUpper(c.JoinType.ValueString())
		}

		hasOpening := false
		if isConfiguredValue(c.HasOpeningParenthesis) {
			hasOpening = c.HasOpeningParenthesis.ValueBool()
		}

		hasClosing := false
		if isConfiguredValue(c.HasClosingParenthesis) {
			hasClosing = c.HasClosingParenthesis.ValueBool()
		}
		crit := client.DeviceGroupCriteriaRepresentationV1{
			AttributeName:         c.AttributeName.ValueString(),
			Operator:              operator,
			AttributeValue:        attributeValue,
			JoinType:              joinType,
			HasOpeningParenthesis: hasOpening,
			HasClosingParenthesis: hasClosing,
		}
		if !c.Order.IsNull() && !c.Order.IsUnknown() {
			crit.Order = int(c.Order.ValueInt64())
		} else {
			crit.Order = idx
		}
		result = append(result, crit)
	}
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].Order < result[j].Order
	})
	return result
}

// flattenDeviceGroupCriteria converts API criteria into Terraform models.
func flattenDeviceGroupCriteria(criteria []client.DeviceGroupCriteriaRepresentationV1, current []DeviceGroupCriteriaModel) []DeviceGroupCriteriaModel {
	if len(criteria) == 0 {
		return nil
	}
	result := make([]DeviceGroupCriteriaModel, len(criteria))
	for i, c := range criteria {
		var prev DeviceGroupCriteriaModel
		if i < len(current) {
			prev = current[i]
		}

		attributeName := types.StringNull()
		if c.AttributeName != "" {
			attributeName = types.StringValue(c.AttributeName)
		}

		operator := types.StringNull()
		if c.Operator != "" {
			operator = types.StringValue(strings.ToLower(c.Operator))
		}

		joinType := types.StringNull()
		if c.JoinType != "" {
			joinType = types.StringValue(strings.ToLower(c.JoinType))
		}
		result[i] = DeviceGroupCriteriaModel{
			Order:                 reconcileOptionalInt(c.Order, prev.Order),
			AttributeName:         attributeName,
			Operator:              operator,
			AttributeValue:        reconcileOptionalString(c.AttributeValue, prev.AttributeValue),
			JoinType:              joinType,
			HasOpeningParenthesis: reconcileOptionalBool(c.HasOpeningParenthesis, prev.HasOpeningParenthesis),
			HasClosingParenthesis: reconcileOptionalBool(c.HasClosingParenthesis, prev.HasClosingParenthesis),
		}
	}
	return result
}

// membersSetToStrings converts a Terraform set of member IDs into a Go slice.
func membersSetToStrings(ctx context.Context, set types.Set) ([]string, diag.Diagnostics) {
	if set.IsNull() || set.IsUnknown() {
		return nil, nil
	}
	var members []string
	diags := set.ElementsAs(ctx, &members, false)
	return members, diags
}

// diffStringSlices returns added/removed values between current and desired sets.
func diffStringSlices(current, desired []string) (added, removed []string) {
	currentSet := make(map[string]struct{}, len(current))
	for _, v := range current {
		currentSet[v] = struct{}{}
	}
	desiredSet := make(map[string]struct{}, len(desired))
	for _, v := range desired {
		desiredSet[v] = struct{}{}
		if _, ok := currentSet[v]; !ok {
			added = append(added, v)
		}
	}
	for _, v := range current {
		if _, ok := desiredSet[v]; !ok {
			removed = append(removed, v)
		}
	}
	sort.Strings(added)
	sort.Strings(removed)
	return
}

// isNotFoundError checks if the error is a 404 not found error.
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	errorStr := err.Error()
	return strings.Contains(errorStr, "status 404") ||
		strings.Contains(errorStr, "was not found") ||
		strings.Contains(errorStr, "NOT_FOUND")
}

// reconcileOptionalBool keeps the current value if not managed, otherwise sets to the API value.
func reconcileOptionalBool(apiValue bool, current types.Bool) types.Bool {
	if isConfiguredValue(current) {
		return types.BoolValue(apiValue)
	}
	return types.BoolNull()
}

// reconcileOptionalInt keeps the current value if not managed, otherwise sets to the API value.
func reconcileOptionalInt(apiValue int, current types.Int64) types.Int64 {
	if isConfiguredValue(current) {
		return types.Int64Value(int64(apiValue))
	}
	return types.Int64Null()
}

// reconcileOptionalString keeps explicit empty strings set by the user while allowing nulls when unset.
func reconcileOptionalString(apiValue string, current types.String) types.String {
	if apiValue == "" {
		if isConfiguredValue(current) && current.ValueString() == "" {
			return current
		}
		return types.StringNull()
	}

	return types.StringValue(apiValue)
}

// stringPointerValue returns a *string for non-null/unknown Terraform strings, preserving empty strings.
func stringPointerValue(v types.String) *string {
	if !isConfiguredValue(v) {
		return nil
	}
	value := v.ValueString()
	return &value
}

// isConfiguredValue reports whether Terraform has a non-null, non-unknown value.
func isConfiguredValue(value interface {
	IsNull() bool
	IsUnknown() bool
}) bool {
	return !value.IsNull() && !value.IsUnknown()
}
