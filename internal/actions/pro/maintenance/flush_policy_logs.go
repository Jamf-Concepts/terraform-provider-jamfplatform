// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package maintenanceactions

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/action"
	actionschema "github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ action.Action = (*FlushPolicyLogsAction)(nil)
var _ action.ActionWithConfigure = (*FlushPolicyLogsAction)(nil)

// FlushPolicyLogsAction flushes the logs for a policy older than the given interval.
type FlushPolicyLogsAction struct {
	maintenanceAction
}

type FlushPolicyLogsActionModel struct {
	PolicyID types.String `tfsdk:"policy_id"`
	Interval types.String `tfsdk:"interval"`
}

func NewFlushPolicyLogsAction() action.Action {
	return &FlushPolicyLogsAction{}
}

func (a *FlushPolicyLogsAction) Metadata(ctx context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pro_flush_policy_logs"
}

func (a *FlushPolicyLogsAction) Schema(ctx context.Context, req action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = actionschema.Schema{
		MarkdownDescription: "Flushes the logs for a policy older than the given interval.",
		Attributes: map[string]actionschema.Attribute{
			"policy_id": actionschema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Jamf Pro policy ID.",
			},
			"interval": actionschema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Flush logs older than this interval, e.g. `Six+Months`.",
				Validators: []validator.String{
					stringvalidator.OneOf(
						"Zero+Days", "Zero+Weeks", "Zero+Months", "Zero+Years",
						"One+Days", "One+Weeks", "One+Months", "One+Years",
						"Two+Days", "Two+Weeks", "Two+Months", "Two+Years",
						"Three+Days", "Three+Weeks", "Three+Months", "Three+Years",
						"Six+Days", "Six+Weeks", "Six+Months", "Six+Years",
					),
				},
			},
		},
	}
}

func (a *FlushPolicyLogsAction) Configure(ctx context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
	a.configure(ctx, req, resp)
}

func (a *FlushPolicyLogsAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	if !a.ensureClassicClient(resp) {
		return
	}

	var data FlushPolicyLogsActionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policyID := data.PolicyID.ValueString()
	interval := data.Interval.ValueString()

	resp.SendProgress(action.InvokeProgressEvent{Message: fmt.Sprintf("Flushing logs older than %s for policy %s", interval, policyID)})

	if err := a.classic.DeleteLogFlushByLogIDInterval(ctx, "policy", policyID, interval); err != nil {
		resp.Diagnostics.AddError(
			"Flush Policy Logs Failed",
			fmt.Sprintf("Unable to flush logs for policy %s: %s", policyID, err),
		)
		return
	}

	resp.SendProgress(action.InvokeProgressEvent{Message: fmt.Sprintf("Flushed logs older than %s for policy %s", interval, policyID)})
}
