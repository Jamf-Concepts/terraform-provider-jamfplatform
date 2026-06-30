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

var _ action.Action = (*UnlockUserAccountAction)(nil)
var _ action.ActionWithConfigure = (*UnlockUserAccountAction)(nil)

// UnlockUserAccountAction unlocks a local user account on a computer.
type UnlockUserAccountAction struct {
	mdmAction
}

type UnlockUserAccountActionModel struct {
	ManagementID types.String `tfsdk:"management_id"`
	SerialNumber types.String `tfsdk:"serial_number"`
	UserName     types.String `tfsdk:"user_name"`
}

func NewUnlockUserAccountAction() action.Action {
	return &UnlockUserAccountAction{}
}

func (a *UnlockUserAccountAction) Metadata(ctx context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_unlock_user_account"
}

func (a *UnlockUserAccountAction) Schema(ctx context.Context, req action.SchemaRequest, resp *action.SchemaResponse) {
	attrs := targetAttributes("computer")
	attrs["user_name"] = actionschema.StringAttribute{
		Optional:            true,
		MarkdownDescription: "Local user account to unlock.",
	}
	resp.Schema = actionschema.Schema{
		MarkdownDescription: "Unlocks a local user account on a computer." + unlockUserAccountPrivileges,
		Attributes:          attrs,
	}
}

func (a *UnlockUserAccountAction) Configure(ctx context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
	a.configure(ctx, req, resp)
}

func (a *UnlockUserAccountAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	if !a.ensureClient(resp) {
		return
	}

	var data UnlockUserAccountActionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	managementID, ok := a.resolveManagementID(ctx, resp, data.ManagementID, data.SerialNumber)
	if !ok {
		return
	}

	command := &pro.UnlockUserAccountCommand{
		CommandType: cmdUnlockUserAccount,
		UserName:    data.UserName.ValueStringPointer(),
	}

	a.sendCommand(ctx, resp, managementID, command, "Unlock user account")
}
