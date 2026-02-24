// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package device_group

import (
	"context"
	"strings"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

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

// flattenDeviceGroupCriteria converts API criteria into Terraform models.
func flattenDeviceGroupCriteria(criteria []client.DeviceGroupCriteriaRepresentationV1, current []DeviceGroupCriteriaModel) []DeviceGroupCriteriaModel {
	if len(criteria) == 0 {
		return nil
	}
	result := make([]DeviceGroupCriteriaModel, len(criteria))
	stateAware := current != nil
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

		var order types.Int64
		if stateAware {
			order = helpers.ReconcileOptionalInt(c.Order, prev.Order)
		} else {
			order = types.Int64Value(int64(c.Order))
		}

		var attributeValue types.String
		if stateAware {
			attributeValue = helpers.ReconcileOptionalString(c.AttributeValue, prev.AttributeValue)
		} else {
			attributeValue = types.StringValue(c.AttributeValue)
		}

		result[i] = DeviceGroupCriteriaModel{
			Order:                 order,
			AttributeName:         attributeName,
			Operator:              operator,
			AttributeValue:        attributeValue,
			JoinType:              joinType,
			HasOpeningParenthesis: helpers.ReconcileOptionalBool(c.HasOpeningParenthesis, prev.HasOpeningParenthesis),
			HasClosingParenthesis: helpers.ReconcileOptionalBool(c.HasClosingParenthesis, prev.HasClosingParenthesis),
		}
	}
	return result
}
