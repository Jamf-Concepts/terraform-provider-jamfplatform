// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package advanced_mobile_device_search

import (
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/criteria"
)

// ValidOperators is the criteria operator vocabulary accepted on advanced mobile
// device search criteria: the full shared canonical set. Mobile-device inventory
// criteria include date attributes (Last Inventory Update, Last Enrollment, …),
// so the full vocabulary applies — unlike
// jamfplatform_pro_advanced_volume_purchasing_content_search, whose content
// attributes use a narrower subset.
var ValidOperators = criteria.Operators
