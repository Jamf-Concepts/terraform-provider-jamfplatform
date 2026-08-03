// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mdmactions

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/action"
	actionschema "github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
)

var _ action.Action = (*DeviceLockAction)(nil)
var _ action.ActionWithConfigure = (*DeviceLockAction)(nil)
var _ action.ActionWithConfigValidators = (*DeviceLockAction)(nil)

// DeviceLockAction locks a computer or mobile device.
type DeviceLockAction struct {
	mdmAction
}

type DeviceLockActionModel struct {
	ManagementID types.String `tfsdk:"management_id"`
	SerialNumber types.String `tfsdk:"serial_number"`
	Message      types.String `tfsdk:"message"`
	PhoneNumber  types.String `tfsdk:"phone_number"`
	Pin          types.String `tfsdk:"pin"`
}

func NewDeviceLockAction() action.Action {
	return &DeviceLockAction{}
}

func (a *DeviceLockAction) Metadata(ctx context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_device_lock"
}

func (a *DeviceLockAction) Schema(ctx context.Context, req action.SchemaRequest, resp *action.SchemaResponse) {
	attrs := targetAttributes("device")
	attrs["message"] = actionschema.StringAttribute{
		Optional:            true,
		MarkdownDescription: "Message to display on the lock screen.",
	}
	attrs["phone_number"] = actionschema.StringAttribute{
		Optional:            true,
		MarkdownDescription: "Phone number to display on the lock screen.",
	}
	attrs["pin"] = actionschema.StringAttribute{
		Optional:            true,
		WriteOnly:           true,
		MarkdownDescription: "Six-character PIN needed to unlock the computer afterwards. Applies to computers only; mobile devices ignore it. Jamf Pro checks the length only, so a six-character non-numeric PIN is accepted here — but macOS expects six digits, so use digits.",
		Validators: []validator.String{
			stringvalidator.LengthBetween(6, 6),
		},
	}
	resp.Schema = actionschema.Schema{
		MarkdownDescription: "Locks a computer or mobile device." + deviceLockPrivileges,
		Attributes:          attrs,
	}
}

func (a *DeviceLockAction) ConfigValidators(ctx context.Context) []action.ConfigValidator {
	return deviceTargetConfigValidators()
}

func (a *DeviceLockAction) Configure(ctx context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
	a.configure(ctx, req, resp)
}

func (a *DeviceLockAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	if !a.ensureClient(resp) {
		return
	}

	var data DeviceLockActionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	managementID, ok := a.resolveManagementID(ctx, resp, data.ManagementID, data.SerialNumber)
	if !ok {
		return
	}

	command := &pro.DeviceLockCommand{
		CommandType: cmdDeviceLock,
		Message:     data.Message.ValueStringPointer(),
		PhoneNumber: data.PhoneNumber.ValueStringPointer(),
		Pin:         data.Pin.ValueStringPointer(),
	}

	a.sendCommand(ctx, resp, managementID, command, "Device lock")
}
