// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mdmactions

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/action"
	actionschema "github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

var _ action.Action = (*ClearPasscodeAction)(nil)
var _ action.ActionWithConfigure = (*ClearPasscodeAction)(nil)

// ClearPasscodeAction clears the passcode on a mobile device.
type ClearPasscodeAction struct {
	mdmAction
}

type ClearPasscodeActionModel struct {
	ManagementID types.String `tfsdk:"management_id"`
	SerialNumber types.String `tfsdk:"serial_number"`
	UnlockToken  types.String `tfsdk:"unlock_token"`
}

func NewClearPasscodeAction() action.Action {
	return &ClearPasscodeAction{}
}

func (a *ClearPasscodeAction) Metadata(ctx context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_clear_passcode"
}

func (a *ClearPasscodeAction) Schema(ctx context.Context, req action.SchemaRequest, resp *action.SchemaResponse) {
	attrs := targetAttributes("mobile device")
	attrs["unlock_token"] = actionschema.StringAttribute{
		Optional:            true,
		MarkdownDescription: "Unlock token for the mobile device. Required for unsupervised devices and looked up automatically when omitted; supervised devices do not need one.",
	}
	resp.Schema = actionschema.Schema{
		MarkdownDescription: "Clears the passcode on a mobile device.",
		Attributes:          attrs,
	}
}

func (a *ClearPasscodeAction) Configure(ctx context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
	a.configure(ctx, req, resp)
}

func (a *ClearPasscodeAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	if !a.ensureClient(resp) {
		return
	}

	var data ClearPasscodeActionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	managementID, ok := a.resolveManagementID(ctx, resp, data.ManagementID, data.SerialNumber)
	if !ok {
		return
	}

	// Prefer an explicit token; otherwise look it up from inventory.
	token := data.UnlockToken.ValueString()
	if !helpers.IsConfiguredValue(data.UnlockToken) {
		resolved, ok := a.resolveUnlockToken(ctx, resp, managementID)
		if !ok {
			return
		}
		token = resolved
	}

	command := &pro.ClearPasscodeCommand{
		CommandType: cmdClearPasscode,
		UnlockToken: token,
	}
	a.sendCommand(ctx, resp, managementID, command, "Clear passcode")
}
