// Copyright 2026 Jamf Software LLC.

package deviceactions

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/action"
	actionschema "github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ action.Action = (*ShutdownAction)(nil)
var _ action.ActionWithConfigure = (*ShutdownAction)(nil)

// ShutdownAction invokes a Jamf Platform shutdown command for a device.
type ShutdownAction struct {
	deviceAction
}

type ShutdownActionModel struct {
	DeviceID types.String `tfsdk:"device_id"`
}

func NewShutdownAction() action.Action {
	return &ShutdownAction{}
}

func (a *ShutdownAction) Metadata(ctx context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_device_shutdown"
}

func (a *ShutdownAction) Schema(ctx context.Context, req action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = actionschema.Schema{
		Description: "Requests that Jamf Platform shut down a managed device.",
		Attributes: map[string]actionschema.Attribute{
			"device_id": actionschema.StringAttribute{
				Required:            true,
				Description:         "UUID of the device to shut down.",
				MarkdownDescription: "UUID of the device to shut down.",
			},
		},
	}
}

func (a *ShutdownAction) Configure(ctx context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
	a.configure(ctx, req, resp)
}

func (a *ShutdownAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	if !a.ensureClient(resp) {
		return
	}

	var data ShutdownActionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deviceID := data.DeviceID.ValueString()

	resp.SendProgress(action.InvokeProgressEvent{Message: fmt.Sprintf("Requesting shutdown for device %s", deviceID)})

	if _, err := a.client.ShutdownDeviceV1(ctx, deviceID); err != nil {
		resp.Diagnostics.AddError(
			"Shutdown Device Failed",
			fmt.Sprintf("Unable to shut down device %s: %s", deviceID, err),
		)
		return
	}

	resp.SendProgress(action.InvokeProgressEvent{Message: fmt.Sprintf("Shutdown request accepted for device %s", deviceID)})
}
