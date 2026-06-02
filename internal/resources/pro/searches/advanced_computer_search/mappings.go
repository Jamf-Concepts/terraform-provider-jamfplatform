// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package advanced_computer_search

import (
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/criteria"
)

// ValidOperators is the criteria operator vocabulary accepted on advanced
// computer search criteria: the full shared canonical set. Computer-inventory
// criteria include date attributes (Last Inventory Update, Last Check-in, …),
// so the two date-window operators that the /usergroups endpoint rejects apply
// here — unlike jamfplatform_pro_user_group.
var ValidOperators = criteria.Operators
