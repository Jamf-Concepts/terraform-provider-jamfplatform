// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mdmactions

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/action"
	actionschema "github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
)

var _ action.Action = (*EnableLostModeAction)(nil)
var _ action.ActionWithConfigure = (*EnableLostModeAction)(nil)

// EnableLostModeAction turns on Lost Mode on a supervised mobile device.
type EnableLostModeAction struct {
	mdmAction
}

type EnableLostModeActionModel struct {
	ManagementID     types.String `tfsdk:"management_id"`
	SerialNumber     types.String `tfsdk:"serial_number"`
	LostModeMessage  types.String `tfsdk:"lost_mode_message"`
	LostModeFootnote types.String `tfsdk:"lost_mode_footnote"`
	LostModePhone    types.String `tfsdk:"lost_mode_phone"`
}

func NewEnableLostModeAction() action.Action {
	return &EnableLostModeAction{}
}

func (a *EnableLostModeAction) Metadata(ctx context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_enable_lost_mode"
}

func (a *EnableLostModeAction) Schema(ctx context.Context, req action.SchemaRequest, resp *action.SchemaResponse) {
	attrs := targetAttributes("mobile device")
	attrs["lost_mode_message"] = actionschema.StringAttribute{
		Optional:            true,
		MarkdownDescription: "Message to display on the Lost Mode lock screen.",
	}
	attrs["lost_mode_footnote"] = actionschema.StringAttribute{
		Optional:            true,
		MarkdownDescription: "Footnote to display on the Lost Mode lock screen.",
	}
	attrs["lost_mode_phone"] = actionschema.StringAttribute{
		Optional:            true,
		MarkdownDescription: "Phone number to display on the Lost Mode lock screen.",
	}
	resp.Schema = actionschema.Schema{
		MarkdownDescription: "Turns on Lost Mode on a supervised mobile device." + enableLostModePrivileges,
		Attributes:          attrs,
	}
}

func (a *EnableLostModeAction) Configure(ctx context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
	a.configure(ctx, req, resp)
}

func (a *EnableLostModeAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	if !a.ensureClient(resp) {
		return
	}

	var data EnableLostModeActionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	managementID, ok := a.resolveManagementID(ctx, resp, data.ManagementID, data.SerialNumber)
	if !ok {
		return
	}

	command := &pro.EnableLostModeCommand{
		CommandType:      cmdEnableLostMode,
		LostModeFootnote: data.LostModeFootnote.ValueStringPointer(),
		LostModeMessage:  data.LostModeMessage.ValueStringPointer(),
		LostModePhone:    data.LostModePhone.ValueStringPointer(),
	}

	a.sendCommand(ctx, resp, managementID, command, "Enable Lost Mode")
}
