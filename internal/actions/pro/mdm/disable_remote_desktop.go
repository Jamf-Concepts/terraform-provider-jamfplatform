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

var _ action.Action = (*DisableRemoteDesktopAction)(nil)
var _ action.ActionWithConfigure = (*DisableRemoteDesktopAction)(nil)

// DisableRemoteDesktopAction disables Remote Desktop (remote management) on a computer.
type DisableRemoteDesktopAction struct {
	mdmAction
}

type DisableRemoteDesktopActionModel struct {
	ManagementID types.String `tfsdk:"management_id"`
	SerialNumber types.String `tfsdk:"serial_number"`
}

func NewDisableRemoteDesktopAction() action.Action {
	return &DisableRemoteDesktopAction{}
}

func (a *DisableRemoteDesktopAction) Metadata(ctx context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_disable_remote_desktop"
}

func (a *DisableRemoteDesktopAction) Schema(ctx context.Context, req action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = actionschema.Schema{
		MarkdownDescription: "Disables Remote Desktop (remote management) on a computer." + disableRemoteDesktopPrivileges,
		Attributes:          targetAttributes("computer"),
	}
}

func (a *DisableRemoteDesktopAction) Configure(ctx context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
	a.configure(ctx, req, resp)
}

func (a *DisableRemoteDesktopAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	if !a.ensureClient(resp) {
		return
	}

	var data DisableRemoteDesktopActionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	managementID, ok := a.resolveManagementID(ctx, resp, data.ManagementID, data.SerialNumber)
	if !ok {
		return
	}

	command := &pro.DisableRemoteDesktopCommand{CommandType: cmdDisableRemoteDesktop}
	a.sendCommand(ctx, resp, managementID, command, "Disable Remote Desktop")
}
