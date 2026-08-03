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

var _ action.Action = (*LogOutUserAction)(nil)
var _ action.ActionWithConfigure = (*LogOutUserAction)(nil)
var _ action.ActionWithConfigValidators = (*LogOutUserAction)(nil)

// LogOutUserAction logs out the current user on a Shared iPad.
type LogOutUserAction struct {
	mdmAction
}

type LogOutUserActionModel struct {
	ManagementID types.String `tfsdk:"management_id"`
	SerialNumber types.String `tfsdk:"serial_number"`
}

func NewLogOutUserAction() action.Action {
	return &LogOutUserAction{}
}

func (a *LogOutUserAction) Metadata(ctx context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_log_out_user"
}

func (a *LogOutUserAction) Schema(ctx context.Context, req action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = actionschema.Schema{
		MarkdownDescription: "Logs out the current user on a Shared iPad." + logOutUserPrivileges,
		Attributes:          targetAttributes("mobile device"),
	}
}

func (a *LogOutUserAction) ConfigValidators(ctx context.Context) []action.ConfigValidator {
	return deviceTargetConfigValidators()
}

func (a *LogOutUserAction) Configure(ctx context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
	a.configure(ctx, req, resp)
}

func (a *LogOutUserAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	if !a.ensureClient(resp) {
		return
	}

	var data LogOutUserActionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	managementID, ok := a.resolveManagementID(ctx, resp, data.ManagementID, data.SerialNumber)
	if !ok {
		return
	}

	command := &pro.LogOutUserCommand{CommandType: cmdLogOutUser}
	a.sendCommand(ctx, resp, managementID, command, "Log out user")
}
