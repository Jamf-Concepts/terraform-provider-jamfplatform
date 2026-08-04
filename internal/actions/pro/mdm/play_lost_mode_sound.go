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

var _ action.Action = (*PlayLostModeSoundAction)(nil)
var _ action.ActionWithConfigure = (*PlayLostModeSoundAction)(nil)
var _ action.ActionWithConfigValidators = (*PlayLostModeSoundAction)(nil)

// PlayLostModeSoundAction plays a sound on a mobile device that is in Lost Mode.
type PlayLostModeSoundAction struct {
	mdmAction
}

type PlayLostModeSoundActionModel struct {
	ManagementIDs types.List `tfsdk:"management_ids"`
	SerialNumbers types.List `tfsdk:"serial_numbers"`
}

func NewPlayLostModeSoundAction() action.Action {
	return &PlayLostModeSoundAction{}
}

func (a *PlayLostModeSoundAction) Metadata(ctx context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_play_lost_mode_sound"
}

func (a *PlayLostModeSoundAction) Schema(ctx context.Context, req action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = actionschema.Schema{
		MarkdownDescription: "Plays a sound on one or more mobile devices that are in Lost Mode." + batchNote + playLostModeSoundPrivileges,
		Attributes:          targetListAttributes("mobile device"),
	}
}

func (a *PlayLostModeSoundAction) ConfigValidators(ctx context.Context) []action.ConfigValidator {
	return deviceTargetListConfigValidators()
}

func (a *PlayLostModeSoundAction) Configure(ctx context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
	a.configure(ctx, req, resp)
}

func (a *PlayLostModeSoundAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	if !a.ensureClient(resp) {
		return
	}

	var data PlayLostModeSoundActionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	managementIDs, ok := a.resolveManagementIDs(ctx, resp, data.ManagementIDs, data.SerialNumbers)
	if !ok {
		return
	}

	command := &pro.PlayLostModeSoundCommand{CommandType: pro.MDMCommandTypePlayLostModeSound}
	a.sendCommandBatch(ctx, resp, managementIDs, command, "Play Lost Mode sound")
}
