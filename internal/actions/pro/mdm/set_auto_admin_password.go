// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mdmactions

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/action"
	actionschema "github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
)

var _ action.Action = (*SetAutoAdminPasswordAction)(nil)
var _ action.ActionWithConfigure = (*SetAutoAdminPasswordAction)(nil)
var _ action.ActionWithConfigValidators = (*SetAutoAdminPasswordAction)(nil)

// SetAutoAdminPasswordAction sets the automatic administrator password on a computer.
type SetAutoAdminPasswordAction struct {
	mdmAction
}

type SetAutoAdminPasswordActionModel struct {
	ManagementID types.String `tfsdk:"management_id"`
	SerialNumber types.String `tfsdk:"serial_number"`
	Guid         types.String `tfsdk:"guid"`
	Password     types.String `tfsdk:"password"`
}

func NewSetAutoAdminPasswordAction() action.Action {
	return &SetAutoAdminPasswordAction{}
}

func (a *SetAutoAdminPasswordAction) Metadata(ctx context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_set_auto_admin_password"
}

func (a *SetAutoAdminPasswordAction) Schema(ctx context.Context, req action.SchemaRequest, resp *action.SchemaResponse) {
	attrs := targetAttributes("computer")
	attrs["guid"] = actionschema.StringAttribute{
		Optional:            true,
		MarkdownDescription: "GUID of the local administrator account whose password is being set.",
	}
	attrs["password"] = actionschema.StringAttribute{
		Optional:            true,
		WriteOnly:           true,
		MarkdownDescription: "New automatic administrator password. Jamf Pro never returns this value, and it must not be empty.",
		Validators: []validator.String{
			stringvalidator.LengthAtLeast(1),
		},
	}
	resp.Schema = actionschema.Schema{
		MarkdownDescription: "Sets the automatic administrator password on a computer." + singleTargetNote + setAutoAdminPasswordPrivileges,
		Attributes:          attrs,
	}
}

func (a *SetAutoAdminPasswordAction) ConfigValidators(ctx context.Context) []action.ConfigValidator {
	return deviceTargetConfigValidators()
}

func (a *SetAutoAdminPasswordAction) Configure(ctx context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
	a.configure(ctx, req, resp)
}

func (a *SetAutoAdminPasswordAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	if !a.ensureClient(resp) {
		return
	}

	var data SetAutoAdminPasswordActionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	managementID, ok := a.resolveManagementID(ctx, resp, data.ManagementID, data.SerialNumber)
	if !ok {
		return
	}

	command := &pro.SetAutoAdminPasswordCommand{
		CommandType: cmdSetAutoAdminPassword,
		Guid:        data.Guid.ValueStringPointer(),
		Password:    data.Password.ValueStringPointer(),
	}

	a.sendCommand(ctx, resp, managementID, command, "Set auto admin password")
}
