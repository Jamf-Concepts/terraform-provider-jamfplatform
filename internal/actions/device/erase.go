// Copyright 2026 Jamf Software LLC.

package deviceactions

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/action"
	actionschema "github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

var _ action.Action = (*EraseAction)(nil)
var _ action.ActionWithConfigure = (*EraseAction)(nil)

// EraseAction invokes a Jamf Platform erase command for a device.
type EraseAction struct {
	deviceAction
}

type EraseActionModel struct {
	DeviceID               types.String `tfsdk:"device_id"`
	PreserveDataPlan       types.Bool   `tfsdk:"preserve_data_plan"`
	DisallowProximitySetup types.Bool   `tfsdk:"disallow_proximity_setup"`
	ClearActivationLock    types.Bool   `tfsdk:"clear_activation_lock"`
	ReturnToService        types.Bool   `tfsdk:"return_to_service"`
	Pin                    types.String `tfsdk:"pin"`
}

func NewEraseAction() action.Action {
	return &EraseAction{}
}

func (a *EraseAction) Metadata(ctx context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_device_erase"
}

func (a *EraseAction) Schema(ctx context.Context, req action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = actionschema.Schema{
		Description: "Requests that Jamf Platform erase a managed device.",
		Attributes: map[string]actionschema.Attribute{
			"device_id": actionschema.StringAttribute{
				Required:            true,
				MarkdownDescription: "UUID of the device to erase.",
			},
			"preserve_data_plan": actionschema.BoolAttribute{
				Optional:    true,
				Description: "Preserve eSIM data plans on supported iPhone or iPad devices.",
			},
			"disallow_proximity_setup": actionschema.BoolAttribute{
				Optional:    true,
				Description: "Disable Proximity Setup on the next reboot for supported mobile devices.",
			},
			"clear_activation_lock": actionschema.BoolAttribute{
				Optional:    true,
				Description: "Clear the Activation Lock for the device (mobile devices only).",
			},
			"return_to_service": actionschema.BoolAttribute{
				Optional:    true,
				Description: "Return the device to service after erase completes (mobile devices only).",
			},
			"pin": actionschema.StringAttribute{
				Optional:    true,
				Description: "Six-digit Find My PIN required for macOS erase operations.",
				Validators: []validator.String{
					stringvalidator.LengthBetween(6, 6),
				},
			},
		},
	}
}

func (a *EraseAction) Configure(ctx context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
	a.configure(ctx, req, resp)
}

func (a *EraseAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	if !a.ensureClient(resp) {
		return
	}

	var data EraseActionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deviceID := data.DeviceID.ValueString()
	request := buildEraseRequestPayload(data)

	resp.SendProgress(action.InvokeProgressEvent{Message: fmt.Sprintf("Requesting erase for device %s", deviceID)})

	commands, err := a.client.EraseDeviceV1(ctx, deviceID, request)
	if err != nil {
		resp.Diagnostics.AddError(
			"Erase Device Failed",
			fmt.Sprintf("Unable to erase device %s: %s", deviceID, err),
		)
		return
	}

	resp.SendProgress(action.InvokeProgressEvent{Message: fmt.Sprintf("Erase request accepted for device %s (%d command(s))", deviceID, len(commands))})
}

func buildEraseRequestPayload(data EraseActionModel) *client.EraseDeviceRequestV1 {
	req := &client.EraseDeviceRequestV1{}
	var hasOptions bool

	if val := helpers.BoolPointerValue(data.PreserveDataPlan); val != nil {
		req.PreserveDataPlan = val
		hasOptions = true
	}
	if val := helpers.BoolPointerValue(data.DisallowProximitySetup); val != nil {
		req.DisallowProximitySetup = val
		hasOptions = true
	}
	if val := helpers.BoolPointerValue(data.ClearActivationLock); val != nil {
		req.ClearActivationLock = val
		hasOptions = true
	}
	if val := helpers.BoolPointerValue(data.ReturnToService); val != nil {
		req.ReturnToService = val
		hasOptions = true
	}
	if val := helpers.StringPointerValue(data.Pin); val != nil {
		req.Pin = val
		hasOptions = true
	}

	if !hasOptions {
		return nil
	}

	return req
}
