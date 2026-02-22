// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package device_group

import (
	"sort"
	"strings"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

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
