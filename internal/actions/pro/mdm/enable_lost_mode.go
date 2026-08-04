// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mdmactions

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/actionvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/action"
	actionschema "github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
)

var _ action.Action = (*EnableLostModeAction)(nil)
var _ action.ActionWithConfigure = (*EnableLostModeAction)(nil)
var _ action.ActionWithConfigValidators = (*EnableLostModeAction)(nil)

// EnableLostModeAction turns on Lost Mode on a supervised mobile device.
type EnableLostModeAction struct {
	mdmAction
}

type EnableLostModeActionModel struct {
	ManagementIDs    types.List   `tfsdk:"management_ids"`
	SerialNumbers    types.List   `tfsdk:"serial_numbers"`
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
	attrs := targetListAttributes("mobile device")
	attrs["lost_mode_message"] = actionschema.StringAttribute{
		Optional:            true,
		MarkdownDescription: "Message to display on the Lost Mode lock screen. Set this or `lost_mode_phone` (or both).",
		Validators: []validator.String{
			stringvalidator.LengthAtLeast(1),
		},
	}
	attrs["lost_mode_footnote"] = actionschema.StringAttribute{
		Optional:            true,
		MarkdownDescription: "Footnote to display on the Lost Mode lock screen. On its own this is not enough — Jamf Pro still requires `lost_mode_message` or `lost_mode_phone`.",
		Validators: []validator.String{
			stringvalidator.LengthAtLeast(1),
		},
	}
	attrs["lost_mode_phone"] = actionschema.StringAttribute{
		Optional:            true,
		MarkdownDescription: "Phone number to display on the Lost Mode lock screen. Set this or `lost_mode_message` (or both).",
		Validators: []validator.String{
			stringvalidator.LengthAtLeast(1),
		},
	}
	resp.Schema = actionschema.Schema{
		MarkdownDescription: "Turns on Lost Mode on one or more supervised mobile devices. At least one of `lost_mode_message` or `lost_mode_phone` must be set — a `lost_mode_footnote` alone is rejected. Every targeted device receives the same message, footnote and phone number." + batchNote + enableLostModePrivileges,
		Attributes:          attrs,
	}
}

func (a *EnableLostModeAction) ConfigValidators(ctx context.Context) []action.ConfigValidator {
	return append(deviceTargetListConfigValidators(),
		// Wire-probed 2026-08-03: Jamf Pro rejects the command with
		// "Either phone number or Lost Mode message must be entered" when both
		// are absent. A footnote alone does not satisfy it.
		actionvalidator.AtLeastOneOf(
			path.MatchRoot("lost_mode_message"),
			path.MatchRoot("lost_mode_phone"),
		),
	)
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

	managementIDs, ok := a.resolveManagementIDs(ctx, resp, data.ManagementIDs, data.SerialNumbers)
	if !ok {
		return
	}

	command := &pro.EnableLostModeCommand{
		CommandType:      cmdEnableLostMode,
		LostModeFootnote: data.LostModeFootnote.ValueStringPointer(),
		LostModeMessage:  data.LostModeMessage.ValueStringPointer(),
		LostModePhone:    data.LostModePhone.ValueStringPointer(),
	}

	a.sendCommandBatch(ctx, resp, managementIDs, command, "Enable Lost Mode")
}
