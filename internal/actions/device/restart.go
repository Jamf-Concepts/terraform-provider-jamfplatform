// Copyright 2026 Jamf Software LLC.

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

var _ action.Action = (*RestartAction)(nil)
var _ action.ActionWithConfigure = (*RestartAction)(nil)

// RestartAction invokes a Jamf Platform restart command for a device.
type RestartAction struct {
	deviceAction
}

// RestartActionModel represents the action config schema.
type RestartActionModel struct {
	DeviceID     types.String `tfsdk:"device_id"`
	SerialNumber types.String `tfsdk:"serial_number"`
}

// NewRestartAction constructs the action.
func NewRestartAction() action.Action {
	return &RestartAction{}
}

func (a *RestartAction) Metadata(ctx context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_device_restart"
}

func (a *RestartAction) Schema(ctx context.Context, req action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = actionschema.Schema{
		MarkdownDescription: "Requests that a device restart. Requires **Device Management Actions API access**.",
		Attributes: map[string]actionschema.Attribute{
			"device_id": actionschema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The ID of the device in UUID format. Provide this or `serial_number`.",
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRelative().AtName("serial_number")),
				},
			},
			"serial_number": actionschema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Device serial number (case-sensitive). Requires **Device Inventory API access** when used. Provide this or `device_id`.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.ConflictsWith(path.MatchRelative().AtName("device_id")),
				},
			},
		},
	}
}

func (a *RestartAction) Configure(ctx context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
	a.configure(ctx, req, resp)
}

func (a *RestartAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	if !a.ensureClient(resp) {
		return
	}

	var data RestartActionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deviceID, ok := a.resolveDeviceIdentifier(ctx, resp, data.DeviceID, data.SerialNumber)
	if !ok {
		return
	}

	resp.SendProgress(action.InvokeProgressEvent{Message: fmt.Sprintf("Requesting restart for device %s", deviceID)})

	if _, err := a.client.RestartDeviceV1(ctx, deviceID); err != nil {
		resp.Diagnostics.AddError(
			"Restart Device Failed",
			fmt.Sprintf("Unable to restart device %s: %s", deviceID, err),
		)
		return
	}

	resp.SendProgress(action.InvokeProgressEvent{Message: fmt.Sprintf("Restart request accepted for device %s", deviceID)})
}
