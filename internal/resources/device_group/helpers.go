// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package device_group

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform"
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/devicegroups"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// refreshDeviceGroupState retrieves the device group from the API and updates the Terraform state.
func (r *DeviceGroupResource) refreshDeviceGroupState(ctx context.Context, id string, model *DeviceGroupResourceModel, manageMembers bool, manageDescription bool, diags *diag.Diagnostics) bool {
	var grp *devicegroups.DeviceGroupReadRepresentationV1
	err := jamfplatform.PollUntil(ctx, deviceGroupCreateRetryDelay, func(pollCtx context.Context) (bool, error) {
		var err error
		grp, err = r.client.GetDeviceGroup(pollCtx, id)
		if err == nil {
			return true, nil
		}
		if !helpers.IsNotFoundError(err) {
			return false, err
		}

		tflog.Debug(pollCtx, "device group not yet available, retrying", map[string]any{
			"id": id,
		})
		return false, nil
	})
	if err != nil {
		diags.AddError("Error reading device group", err.Error())
		return false
	}

	var members []string
	if strings.EqualFold(grp.GroupType, "STATIC") && manageMembers {
		var err error
		members, err = r.client.ListDeviceGroupMembers(ctx, grp.ID)
		if err != nil {
			diags.AddError("Error reading device group members", err.Error())
			return false
		}
	}

	diags.Append(assignDeviceGroupModel(ctx, model, grp, members, manageMembers, manageDescription)...)
	return !diags.HasError()
}

// validateDeviceGroupPlan checks that the plan is valid based on group type.
func validateDeviceGroupPlan(plan *DeviceGroupResourceModel) error {
	groupType := strings.ToLower(plan.GroupType.ValueString())

	switch groupType {
	case "smart":
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
	slices.Sort(added)
	slices.Sort(removed)
	return
}
