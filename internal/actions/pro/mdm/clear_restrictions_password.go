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

var _ action.Action = (*ClearRestrictionsPasswordAction)(nil)
var _ action.ActionWithConfigure = (*ClearRestrictionsPasswordAction)(nil)
var _ action.ActionWithConfigValidators = (*ClearRestrictionsPasswordAction)(nil)

// ClearRestrictionsPasswordAction clears the Screen Time (restrictions) passcode on a mobile device.
type ClearRestrictionsPasswordAction struct {
	mdmAction
}

type ClearRestrictionsPasswordActionModel struct {
	ManagementIDs types.List `tfsdk:"management_ids"`
	SerialNumbers types.List `tfsdk:"serial_numbers"`
}

func NewClearRestrictionsPasswordAction() action.Action {
	return &ClearRestrictionsPasswordAction{}
}

func (a *ClearRestrictionsPasswordAction) Metadata(ctx context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_clear_restrictions_password"
}

func (a *ClearRestrictionsPasswordAction) Schema(ctx context.Context, req action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = actionschema.Schema{
		MarkdownDescription: "Clears the Screen Time (restrictions) passcode on one or more mobile devices." + batchNote + clearRestrictionsPasswordPrivileges,
		Attributes:          targetListAttributes("mobile device"),
	}
}

func (a *ClearRestrictionsPasswordAction) ConfigValidators(ctx context.Context) []action.ConfigValidator {
	return deviceTargetListConfigValidators()
}

func (a *ClearRestrictionsPasswordAction) Configure(ctx context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
	a.configure(ctx, req, resp)
}

func (a *ClearRestrictionsPasswordAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	if !a.ensureClient(resp) {
		return
	}

	var data ClearRestrictionsPasswordActionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	managementIDs, ok := a.resolveManagementIDs(ctx, resp, data.ManagementIDs, data.SerialNumbers)
	if !ok {
		return
	}

	command := &pro.ClearRestrictionsPasswordCommand{CommandType: pro.MDMCommandTypeClearRestrictionsPassword}
	a.sendCommandBatch(ctx, resp, managementIDs, command, "Clear restrictions password")
}
