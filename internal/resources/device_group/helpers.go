// Copyright 2025 Jamf Software LLC.

package device_group

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
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
	attempt := 0
	var lastErr error
	return helpers.PollUntil(ctx, deviceGroupCreateRetryDelay, func(pollCtx context.Context) (bool, error) {
		attempt++
		_, err := r.client.GetDeviceGroupByIDV1(pollCtx, id)
		if err == nil {
			return true, nil
		}
		if !helpers.IsNotFoundError(err) {
			return false, err
		}

		lastErr = err
		if attempt >= deviceGroupCreateMaxAttempts {
			return false, fmt.Errorf("device group %s not yet available after %d attempts: %w", id, deviceGroupCreateMaxAttempts, lastErr)
		}

		tflog.Debug(pollCtx, "device group not yet available, retrying", map[string]interface{}{
			"id":      id,
			"attempt": attempt,
		})
		return false, nil
	})
}

// assignDeviceGroupModel maps API representation to Terraform model, respecting managed fields.
func assignDeviceGroupModel(ctx context.Context, model *DeviceGroupResourceModel, grp *client.DeviceGroupReadRepresentationV1, members []string, manageMembers bool, manageDescription bool) diag.Diagnostics {
	var diags diag.Diagnostics

	prevDescription := model.Description
	prevCriteria := model.Criteria
	model.ID = types.StringValue(grp.ID)
	if grp.Name == "" {
		model.Name = types.StringNull()
	} else {
		model.Name = types.StringValue(grp.Name)
	}
	if manageDescription {
		model.Description = helpers.ReconcileOptionalString(grp.Description, prevDescription)
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
		if helpers.IsConfiguredValue(plan.Members) {
			return fmt.Errorf("members cannot be set for smart groups")
		}
	case "static":
		if len(plan.Criteria) > 0 {
			return fmt.Errorf("criteria cannot be set for static groups")
		}
		if plan.Members.IsUnknown() {
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
		if !helpers.IsConfiguredValue(c.AttributeName) {
			continue
		}

		operator := ""
		if helpers.IsConfiguredValue(c.Operator) {
			operator = strings.ToUpper(c.Operator.ValueString())
		}

		attributeValue := ""
		if helpers.IsConfiguredValue(c.AttributeValue) {
			attributeValue = c.AttributeValue.ValueString()
		}

		joinType := ""
		if helpers.IsConfiguredValue(c.JoinType) {
			joinType = strings.ToUpper(c.JoinType.ValueString())
		}

		hasOpening := false
		if helpers.IsConfiguredValue(c.HasOpeningParenthesis) {
			hasOpening = c.HasOpeningParenthesis.ValueBool()
		}

		hasClosing := false
		if helpers.IsConfiguredValue(c.HasClosingParenthesis) {
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
		if helpers.IsConfiguredValue(c.Order) {
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
			Order:                 helpers.ReconcileOptionalInt(c.Order, prev.Order),
			AttributeName:         attributeName,
			Operator:              operator,
			AttributeValue:        helpers.ReconcileOptionalString(c.AttributeValue, prev.AttributeValue),
			JoinType:              joinType,
			HasOpeningParenthesis: helpers.ReconcileOptionalBool(c.HasOpeningParenthesis, prev.HasOpeningParenthesis),
			HasClosingParenthesis: helpers.ReconcileOptionalBool(c.HasClosingParenthesis, prev.HasClosingParenthesis),
		}
	}
	return result
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
