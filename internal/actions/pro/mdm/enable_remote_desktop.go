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

var _ action.Action = (*EnableRemoteDesktopAction)(nil)
var _ action.ActionWithConfigure = (*EnableRemoteDesktopAction)(nil)

// EnableRemoteDesktopAction enables Remote Desktop (remote management) on a computer.
type EnableRemoteDesktopAction struct {
	mdmAction
}

type EnableRemoteDesktopActionModel struct {
	ManagementID types.String `tfsdk:"management_id"`
	SerialNumber types.String `tfsdk:"serial_number"`
}

func NewEnableRemoteDesktopAction() action.Action {
	return &EnableRemoteDesktopAction{}
}

func (a *EnableRemoteDesktopAction) Metadata(ctx context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_enable_remote_desktop"
}

func (a *EnableRemoteDesktopAction) Schema(ctx context.Context, req action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = actionschema.Schema{
		MarkdownDescription: "Enables Remote Desktop (remote management) on a computer.",
		Attributes:          targetAttributes("computer"),
	}
}

func (a *EnableRemoteDesktopAction) Configure(ctx context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
	a.configure(ctx, req, resp)
}

func (a *EnableRemoteDesktopAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	if !a.ensureClient(resp) {
		return
	}

	var data EnableRemoteDesktopActionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	managementID, ok := a.resolveManagementID(ctx, resp, data.ManagementID, data.SerialNumber)
	if !ok {
		return
	}

	command := &pro.EnableRemoteDesktopCommand{CommandType: cmdEnableRemoteDesktop}
	a.sendCommand(ctx, resp, managementID, command, "Enable Remote Desktop")
}
