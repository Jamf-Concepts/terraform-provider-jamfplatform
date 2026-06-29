// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package deviceactions

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/action"
	actionschema "github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ action.Action = (*UnmanageAction)(nil)
var _ action.ActionWithConfigure = (*UnmanageAction)(nil)

// UnmanageAction invokes a Jamf Platform unmanage command for a device.
type UnmanageAction struct {
	deviceAction
}

type UnmanageActionModel struct {
	DeviceID     types.String `tfsdk:"device_id"`
	SerialNumber types.String `tfsdk:"serial_number"`
}

func NewUnmanageAction() action.Action {
	return &UnmanageAction{}
}

func (a *UnmanageAction) Metadata(ctx context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_device_unmanage"
}

func (a *UnmanageAction) Schema(ctx context.Context, req action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = actionschema.Schema{
		MarkdownDescription: "Removes remote management from a device. Requires **Device Management Actions API access**." + unmanageDevicePrivileges,
		Attributes: map[string]actionschema.Attribute{
			"device_id": actionschema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Jamf Pro Management ID. Provide this or `serial_number`.",
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("serial_number")),
				},
			},
			"serial_number": actionschema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Device serial number (case-sensitive). Requires **Device Inventory API access** when used. Provide this or `device_id`.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("device_id")),
				},
			},
		},
	}
}

func (a *UnmanageAction) Configure(ctx context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
	a.configure(ctx, req, resp)
}

func (a *UnmanageAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	if !a.ensureClient(resp) {
		return
	}

	var data UnmanageActionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deviceID, ok := a.resolveDeviceIdentifier(ctx, resp, data.DeviceID, data.SerialNumber)
	if !ok {
		return
	}

	resp.SendProgress(action.InvokeProgressEvent{Message: fmt.Sprintf("Requesting unmanage for device %s", deviceID)})

	if _, err := a.actions.UnmanageDevice(ctx, deviceID); err != nil {
		resp.Diagnostics.AddError(
			"Unmanage Device Failed",
			fmt.Sprintf("Unable to unmanage device %s: %s", deviceID, err),
		)
		return
	}

	resp.SendProgress(action.InvokeProgressEvent{Message: fmt.Sprintf("Unmanage request accepted for device %s", deviceID)})
}
