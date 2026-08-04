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

var _ action.Action = (*CancelEnhancedLogCollectionAction)(nil)
var _ action.ActionWithConfigure = (*CancelEnhancedLogCollectionAction)(nil)
var _ action.ActionWithConfigValidators = (*CancelEnhancedLogCollectionAction)(nil)

// CancelEnhancedLogCollectionAction stops an in-progress AppleCare enhanced log
// collection session on one or more devices.
//
// Unlike its trigger counterpart this command carries no payload, so there is no
// token to scope and nothing that could be misapplied across a batch.
type CancelEnhancedLogCollectionAction struct {
	mdmAction
}

type CancelEnhancedLogCollectionActionModel struct {
	ManagementIDs types.List `tfsdk:"management_ids"`
	SerialNumbers types.List `tfsdk:"serial_numbers"`
}

func NewCancelEnhancedLogCollectionAction() action.Action {
	return &CancelEnhancedLogCollectionAction{}
}

func (a *CancelEnhancedLogCollectionAction) Metadata(ctx context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_cancel_enhanced_log_collection"
}

func (a *CancelEnhancedLogCollectionAction) Schema(ctx context.Context, req action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = actionschema.Schema{
		MarkdownDescription: "Stops an in-progress AppleCare enhanced log collection session on one or more devices. The outcome for each device is reported in that device's management history." +
			enhancedLogCollectionNote + batchNote + cancelEnhancedLogCollectionPrivileges,
		Attributes: targetListAttributes("device"),
	}
}

func (a *CancelEnhancedLogCollectionAction) ConfigValidators(ctx context.Context) []action.ConfigValidator {
	return deviceTargetListConfigValidators()
}

func (a *CancelEnhancedLogCollectionAction) Configure(ctx context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
	a.configureWithFloor(ctx, req, resp, minJamfProVersionEnhancedLogCollection, "jamfplatform_pro_cancel_enhanced_log_collection")
}

func (a *CancelEnhancedLogCollectionAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	if !a.ensureClient(resp) {
		return
	}

	var data CancelEnhancedLogCollectionActionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	managementIDs, ok := a.resolveManagementIDs(ctx, resp, data.ManagementIDs, data.SerialNumbers)
	if !ok {
		return
	}

	command := &pro.CancelEnhancedLogCollectionCommand{
		CommandType: pro.MDMCommandTypeCancelEnhancedLogCollection,
	}
	a.sendCommandBatch(ctx, resp, managementIDs, command, "Cancel Enhanced Log Collection")
}
