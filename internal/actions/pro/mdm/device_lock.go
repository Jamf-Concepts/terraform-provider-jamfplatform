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
	ManagementIDs types.List   `tfsdk:"management_ids"`
	SerialNumbers types.List   `tfsdk:"serial_numbers"`
	Message       types.String `tfsdk:"message"`
	PhoneNumber   types.String `tfsdk:"phone_number"`
	Pin           types.String `tfsdk:"pin"`
}

func NewDeviceLockAction() action.Action {
	return &DeviceLockAction{}
}

func (a *DeviceLockAction) Metadata(ctx context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_device_lock"
}

func (a *DeviceLockAction) Schema(ctx context.Context, req action.SchemaRequest, resp *action.SchemaResponse) {
	attrs := targetListAttributes("device")
	attrs["message"] = actionschema.StringAttribute{
		Optional:            true,
		MarkdownDescription: "Message to display on the lock screen.",
	}
	attrs["phone_number"] = actionschema.StringAttribute{
		Optional:            true,
		MarkdownDescription: "Phone number to display on the lock screen.",
	}
	// Deliberately neither WriteOnly nor Sensitive — see secretAttrNote.
	attrs["pin"] = actionschema.StringAttribute{
		Optional:            true,
		MarkdownDescription: "Six-character PIN needed to unlock the computer afterwards. Applies to computers only; mobile devices ignore it. Jamf Pro checks the length only, so a six-character non-numeric PIN is accepted here — but macOS expects six digits, so use digits." + secretAttrNote,
		Validators: []validator.String{
			stringvalidator.LengthBetween(6, 6),
		},
	}
	resp.Schema = actionschema.Schema{
		MarkdownDescription: "Locks one or more computers or mobile devices. Every targeted device receives the same `pin`, `message` and `phone_number`." + batchNote + deviceLockPrivileges,
		Attributes:          attrs,
	}
}

func (a *DeviceLockAction) ConfigValidators(ctx context.Context) []action.ConfigValidator {
	return deviceTargetListConfigValidators()
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

	managementIDs, ok := a.resolveManagementIDs(ctx, resp, data.ManagementIDs, data.SerialNumbers)
	if !ok {
		return
	}

	command := &pro.DeviceLockCommand{
		CommandType: pro.MDMCommandTypeDeviceLock,
		Message:     data.Message.ValueStringPointer(),
		PhoneNumber: data.PhoneNumber.ValueStringPointer(),
		Pin:         data.Pin.ValueStringPointer(),
	}

	a.sendCommandBatch(ctx, resp, managementIDs, command, "Device lock")
}
