// Copyright 2026 Jamf Software LLC.

package deviceactions

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/action"
	actionschema "github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ action.Action = (*UnmanageAction)(nil)
var _ action.ActionWithConfigure = (*UnmanageAction)(nil)

// UnmanageAction invokes a Jamf Platform unmanage command for a device.
type UnmanageAction struct {
	deviceAction
}

type UnmanageActionModel struct {
	DeviceID types.String `tfsdk:"device_id"`
}

func NewUnmanageAction() action.Action {
	return &UnmanageAction{}
}

func (a *UnmanageAction) Metadata(ctx context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_device_unmanage"
}

func (a *UnmanageAction) Schema(ctx context.Context, req action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = actionschema.Schema{
		Description: "Requests that Jamf Platform remove remote management from a device.",
		Attributes: map[string]actionschema.Attribute{
			"device_id": actionschema.StringAttribute{
				Required:            true,
				Description:         "UUID of the device to unmanage.",
				MarkdownDescription: "UUID of the device to unmanage.",
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

	deviceID := data.DeviceID.ValueString()

	resp.SendProgress(action.InvokeProgressEvent{Message: fmt.Sprintf("Requesting unmanage for device %s", deviceID)})

	if _, err := a.client.UnmanageDeviceV1(ctx, deviceID); err != nil {
		resp.Diagnostics.AddError(
			"Unmanage Device Failed",
			fmt.Sprintf("Unable to unmanage device %s: %s", deviceID, err),
		)
		return
	}

	resp.SendProgress(action.InvokeProgressEvent{Message: fmt.Sprintf("Unmanage request accepted for device %s", deviceID)})
}
