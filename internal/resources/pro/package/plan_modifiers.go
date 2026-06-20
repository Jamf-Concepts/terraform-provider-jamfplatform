// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package pkg

import (
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/planmodifiers"
)

// resetIfSourceChangedString is a package-local alias for the shared
// planmodifiers.ResetIfSourceChangedString.
func resetIfSourceChangedString(sourcePaths ...path.Expression) planmodifier.String {
	return planmodifiers.ResetIfSourceChangedString(sourcePaths...)
}
