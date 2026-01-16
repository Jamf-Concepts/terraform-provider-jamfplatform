// Copyright 2026 Jamf Software LLC.

package deviceactions

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/action"
	actionschema "github.com/hashicorp/terraform-plugin-framework/action/schema"
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
	DeviceID types.String `tfsdk:"device_id"`
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
		Description: "Requests that Jamf Platform restart a managed device.",
		Attributes: map[string]actionschema.Attribute{
			"device_id": actionschema.StringAttribute{
				Required:            true,
				Description:         "UUID of the device to restart.",
				MarkdownDescription: "UUID of the device to restart.",
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

	deviceID := data.DeviceID.ValueString()

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
