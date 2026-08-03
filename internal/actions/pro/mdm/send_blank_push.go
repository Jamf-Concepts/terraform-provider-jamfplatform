// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mdmactions

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/actionvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/action"
	actionschema "github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
)

var _ action.Action = (*SendBlankPushAction)(nil)
var _ action.ActionWithConfigure = (*SendBlankPushAction)(nil)
var _ action.ActionWithConfigValidators = (*SendBlankPushAction)(nil)

// SendBlankPushAction sends a blank push notification to one or more devices.
type SendBlankPushAction struct {
	mdmAction
}

type SendBlankPushActionModel struct {
	ManagementIDs types.List `tfsdk:"management_ids"`
	SerialNumbers types.List `tfsdk:"serial_numbers"`
}

func NewSendBlankPushAction() action.Action {
	return &SendBlankPushAction{}
}

func (a *SendBlankPushAction) Metadata(ctx context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_send_blank_push"
}

func (a *SendBlankPushAction) Schema(ctx context.Context, req action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = actionschema.Schema{
		MarkdownDescription: "Sends a blank push notification to one or more devices to prompt them to check in." + sendBlankPushPrivileges,
		Attributes: map[string]actionschema.Attribute{
			"management_ids": actionschema.ListAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Jamf Pro Management IDs of the devices to push. Set this and/or `serial_numbers`.",
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
					listvalidator.ValueStringsAre(stringvalidator.LengthAtLeast(1)),
				},
			},
			"serial_numbers": actionschema.ListAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Serial numbers of the devices to push (case-sensitive). Set this and/or `management_ids`.",
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
					listvalidator.ValueStringsAre(stringvalidator.LengthAtLeast(1)),
				},
			},
		},
	}
}

func (a *SendBlankPushAction) ConfigValidators(ctx context.Context) []action.ConfigValidator {
	return []action.ConfigValidator{
		actionvalidator.AtLeastOneOf(
			path.MatchRoot("management_ids"),
			path.MatchRoot("serial_numbers"),
		),
	}
}

func (a *SendBlankPushAction) Configure(ctx context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
	a.configure(ctx, req, resp)
}

func (a *SendBlankPushAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	if !a.ensureClient(resp) {
		return
	}

	var data SendBlankPushActionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var ids []string
	if !data.ManagementIDs.IsNull() && !data.ManagementIDs.IsUnknown() {
		resp.Diagnostics.Append(data.ManagementIDs.ElementsAs(ctx, &ids, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	if !data.SerialNumbers.IsNull() && !data.SerialNumbers.IsUnknown() {
		var serials []string
		resp.Diagnostics.Append(data.SerialNumbers.ElementsAs(ctx, &serials, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		for _, serial := range serials {
			resp.SendProgress(action.InvokeProgressEvent{Message: fmt.Sprintf("Resolving serial number %s", serial)})
			id, err := a.devices.ResolveDeviceIDBySerialNumber(ctx, serial)
			if err != nil {
				resp.Diagnostics.AddError(
					"Device Lookup Failed",
					fmt.Sprintf("Unable to resolve serial number %s to a management id: %s", serial, err),
				)
				return
			}
			ids = append(ids, id)
		}
	}

	if len(ids) == 0 {
		resp.Diagnostics.AddError(
			"Missing Device Identifier",
			"Specify at least one of management_ids or serial_numbers.",
		)
		return
	}

	resp.SendProgress(action.InvokeProgressEvent{Message: fmt.Sprintf("Sending blank push to %d device(s)", len(ids))})

	out, err := a.client.SendMdmBlankPushV2(ctx, &pro.BlankPushRequest{ClientManagementIds: ids})
	if err != nil {
		resp.Diagnostics.AddError(
			"Blank Push Failed",
			fmt.Sprintf("Unable to send blank push: %s", err),
		)
		return
	}

	if out != nil && len(out.ErrorUuids) > 0 {
		resp.Diagnostics.AddWarning(
			"Blank Push Partially Failed",
			fmt.Sprintf("Some devices did not accept the blank push: %v", out.ErrorUuids),
		)
	}

	resp.SendProgress(action.InvokeProgressEvent{Message: fmt.Sprintf("Blank push accepted for %d device(s)", len(ids))})
}
