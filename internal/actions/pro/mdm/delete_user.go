// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mdmactions

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/action"
	actionschema "github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
)

var _ action.Action = (*DeleteUserAction)(nil)
var _ action.ActionWithConfigure = (*DeleteUserAction)(nil)
var _ action.ActionWithConfigValidators = (*DeleteUserAction)(nil)

// DeleteUserAction removes a user account from a Shared iPad.
type DeleteUserAction struct {
	mdmAction
}

type DeleteUserActionModel struct {
	ManagementIDs  types.List   `tfsdk:"management_ids"`
	SerialNumbers  types.List   `tfsdk:"serial_numbers"`
	UserName       types.String `tfsdk:"user_name"`
	DeleteAllUsers types.Bool   `tfsdk:"delete_all_users"`
	ForceDeletion  types.Bool   `tfsdk:"force_deletion"`
}

func NewDeleteUserAction() action.Action {
	return &DeleteUserAction{}
}

func (a *DeleteUserAction) Metadata(ctx context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_delete_user"
}

func (a *DeleteUserAction) Schema(ctx context.Context, req action.SchemaRequest, resp *action.SchemaResponse) {
	attrs := targetListAttributes("mobile device")
	attrs["user_name"] = actionschema.StringAttribute{
		Optional:            true,
		MarkdownDescription: "User account to remove from the Shared iPad. Set this or `delete_all_users`, not both.",
		Validators: []validator.String{
			stringvalidator.LengthAtLeast(1),
			stringvalidator.ConflictsWith(path.MatchRoot("delete_all_users")),
		},
	}
	attrs["delete_all_users"] = actionschema.BoolAttribute{
		Optional:            true,
		MarkdownDescription: "Remove every user account from the Shared iPad. Set this or `user_name`, not both.",
	}
	attrs["force_deletion"] = actionschema.BoolAttribute{
		Optional:            true,
		MarkdownDescription: "Force removal even when the account has unsynced data.",
	}
	resp.Schema = actionschema.Schema{
		MarkdownDescription: "Removes a user account from one or more Shared iPads. Name a single account with `user_name`, or clear them all with `delete_all_users`; either way the same choice applies to every targeted device." + batchNote + deleteUserPrivileges,
		Attributes:          attrs,
	}
}

func (a *DeleteUserAction) ConfigValidators(ctx context.Context) []action.ConfigValidator {
	return deviceTargetListConfigValidators()
}

func (a *DeleteUserAction) Configure(ctx context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
	a.configure(ctx, req, resp)
}

func (a *DeleteUserAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	if !a.ensureClient(resp) {
		return
	}

	var data DeleteUserActionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	managementIDs, ok := a.resolveManagementIDs(ctx, resp, data.ManagementIDs, data.SerialNumbers)
	if !ok {
		return
	}

	command := &pro.DeleteUserCommand{
		CommandType:    pro.MDMCommandTypeDeleteUser,
		DeleteAllUsers: data.DeleteAllUsers.ValueBoolPointer(),
		ForceDeletion:  data.ForceDeletion.ValueBoolPointer(),
		UserName:       data.UserName.ValueStringPointer(),
	}

	a.sendCommandBatch(ctx, resp, managementIDs, command, "Delete user")
}
