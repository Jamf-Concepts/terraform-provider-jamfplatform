// Copyright 2026 Jamf Software LLC.

package deviceactions

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/boolvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/action"
	actionschema "github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
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
	SerialNumber           types.String `tfsdk:"serial_number"`
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
		MarkdownDescription: "Requests that a device erase its content and settings. Requires **Device Management Actions API access**.",
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
			"preserve_data_plan": actionschema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Preserve the data plan on an iPhone or iPad with eSIM functionality, if one exists. Applies to mobile devices only.",
				Validators: []validator.Bool{
					boolvalidator.ConflictsWith(path.MatchRelative().AtName("pin")),
				},
			},
			"disallow_proximity_setup": actionschema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Disable Proximity Setup on the next reboot and skip the pane in Setup Assistant. Applies to mobile devices only.",
				Validators: []validator.Bool{
					boolvalidator.ConflictsWith(path.MatchRelative().AtName("pin")),
				},
			},
			"clear_activation_lock": actionschema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Clear the activation lock on the device. Applies to mobile devices only.",
				Validators: []validator.Bool{
					boolvalidator.ConflictsWith(path.MatchRelative().AtName("pin")),
				},
			},
			"return_to_service": actionschema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "The device will be returned to service after the erase is complete. Applies to mobile devices only.",
				Validators: []validator.Bool{
					boolvalidator.ConflictsWith(path.MatchRelative().AtName("pin")),
				},
			},
			"pin": actionschema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The six-character PIN for Find My. Applies to computers only.",
				Validators: []validator.String{
					stringvalidator.LengthBetween(6, 6),
					stringvalidator.ConflictsWith(
						path.MatchRelative().AtName("preserve_data_plan"),
						path.MatchRelative().AtName("disallow_proximity_setup"),
						path.MatchRelative().AtName("clear_activation_lock"),
						path.MatchRelative().AtName("return_to_service"),
					),
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

	deviceID, ok := a.resolveDeviceIdentifier(ctx, resp, data.DeviceID, data.SerialNumber)
	if !ok {
		return
	}
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
