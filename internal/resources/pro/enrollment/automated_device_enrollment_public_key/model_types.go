// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package automated_device_enrollment_public_key

import (
	datasourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// AutomatedDeviceEnrollmentPublicKeyDataSourceModel is the Terraform model
// for the singleton `jamfplatform_pro_automated_device_enrollment_public_key`
// data source. The Jamf Pro tenant exposes exactly one ADE public key, so the
// data source takes no input attributes — `id` is a fixed literal and
// `public_key` is the base64-encoded body returned by the SDK.
type AutomatedDeviceEnrollmentPublicKeyDataSourceModel struct {
	ID        types.String             `tfsdk:"id"`
	PublicKey types.String             `tfsdk:"public_key"`
	Timeouts  datasourceTimeouts.Value `tfsdk:"timeouts"`
}
