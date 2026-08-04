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

var _ action.Action = (*TriggerEnhancedLogCollectionAction)(nil)
var _ action.ActionWithConfigure = (*TriggerEnhancedLogCollectionAction)(nil)
var _ action.ActionWithConfigValidators = (*TriggerEnhancedLogCollectionAction)(nil)

// enhancedLogCollectionNote is shared by the trigger and cancel actions. Both
// carry the same two caveats, and stating them in the rendered docs matters more
// than usual here: the command is accepted by Jamf Pro but silently does nothing
// on a device whose OS is too old, so a user with no visible error needs to know
// where to look.
const enhancedLogCollectionNote = "\n\n" +
	"Requires Jamf Pro 11.30 or later. Enhanced log collection is an Apple feature " +
	"available on iOS, iPadOS, tvOS and macOS 27.0 or later — Jamf Pro will queue the " +
	"command for any targeted device, but a device on an earlier OS cannot act on it."

// triggerTokenScopeNote documents the one thing Apple's own documentation leaves
// open, so a user batching this action knows what they are relying on.
const triggerTokenScopeNote = "\n\n" +
	"The same `apple_care_token` is sent to every targeted device. Apple documents the " +
	"token as authorising \"the enhanced log collection session\" and states that it is " +
	"issued as part of an AppleCare ticket, but does not say whether one token is valid " +
	"for more than one device. If AppleCare issued the token for a single device, target " +
	"only that device: the other devices will reject the command at the device rather " +
	"than at Jamf Pro, so the invocation itself still succeeds and the failures appear " +
	"in each device's management history."

// TriggerEnhancedLogCollectionAction starts an AppleCare enhanced log collection
// session on one or more devices.
type TriggerEnhancedLogCollectionAction struct {
	mdmAction
}

type TriggerEnhancedLogCollectionActionModel struct {
	ManagementIDs  types.List   `tfsdk:"management_ids"`
	SerialNumbers  types.List   `tfsdk:"serial_numbers"`
	AppleCareToken types.String `tfsdk:"apple_care_token"`
}

func NewTriggerEnhancedLogCollectionAction() action.Action {
	return &TriggerEnhancedLogCollectionAction{}
}

func (a *TriggerEnhancedLogCollectionAction) Metadata(ctx context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_trigger_enhanced_log_collection"
}

func (a *TriggerEnhancedLogCollectionAction) Schema(ctx context.Context, req action.SchemaRequest, resp *action.SchemaResponse) {
	attrs := targetListAttributes("device")
	// Deliberately neither WriteOnly nor Sensitive — see secretAttrNote.
	attrs["apple_care_token"] = actionschema.StringAttribute{
		Required:            true,
		MarkdownDescription: "AppleCare token authorising the enhanced log collection session. Obtained from Apple as part of an AppleCare ticket; AppleCare issues either an interactive token (the device prompts the user for consent) or a non-interactive one (the device shows a notification and uploads in the background)." + secretAttrNote,
		Validators: []validator.String{
			stringvalidator.LengthAtLeast(1),
		},
	}

	resp.Schema = actionschema.Schema{
		MarkdownDescription: "Starts an AppleCare enhanced log collection session on one or more devices, so diagnostic logs are collected and uploaded to Apple for an AppleCare escalation." +
			enhancedLogCollectionNote + triggerTokenScopeNote + batchNote + triggerEnhancedLogCollectionPrivileges,
		Attributes: attrs,
	}
}

func (a *TriggerEnhancedLogCollectionAction) ConfigValidators(ctx context.Context) []action.ConfigValidator {
	return deviceTargetListConfigValidators()
}

func (a *TriggerEnhancedLogCollectionAction) Configure(ctx context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
	a.configureWithFloor(ctx, req, resp, minJamfProVersionEnhancedLogCollection, "jamfplatform_pro_trigger_enhanced_log_collection")
}

func (a *TriggerEnhancedLogCollectionAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	if !a.ensureClient(resp) {
		return
	}

	var data TriggerEnhancedLogCollectionActionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	managementIDs, ok := a.resolveManagementIDs(ctx, resp, data.ManagementIDs, data.SerialNumbers)
	if !ok {
		return
	}

	command := &pro.TriggerEnhancedLogCollectionCommand{
		CommandType:    pro.MDMCommandTypeTriggerEnhancedLogCollection,
		AppleCareToken: data.AppleCareToken.ValueString(),
	}
	a.sendCommandBatch(ctx, resp, managementIDs, command, "Trigger Enhanced Log Collection")
}
