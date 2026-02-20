// Copyright 2025 Jamf Software LLC.

package device_group

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.ResourceWithUpgradeState = &DeviceGroupResource{}

// UpgradeState returns state upgraders for migrating between schema versions.
func (r *DeviceGroupResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: deviceGroupSchemaV0(),
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var old DeviceGroupResourceModel
				resp.Diagnostics.Append(req.State.Get(ctx, &old)...)
				if resp.Diagnostics.HasError() {
					return
				}
				resp.Diagnostics.Append(resp.State.Set(ctx, &old)...)
			},
		},
	}
}

// deviceGroupSchemaV0 returns the v0 schema where criteria was a ListNestedBlock.
func deviceGroupSchemaV0() *schema.Schema {
	return &schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id":           schema.StringAttribute{Computed: true},
			"name":         schema.StringAttribute{Required: true},
			"description":  schema.StringAttribute{Optional: true},
			"device_type":  schema.StringAttribute{Required: true},
			"group_type":   schema.StringAttribute{Required: true},
			"members":      schema.SetAttribute{Optional: true, ElementType: types.StringType},
			"member_count": schema.Int64Attribute{Computed: true},
			"timeouts":     schema.ObjectAttribute{Optional: true, AttributeTypes: deviceGroupTimeoutAttributeTypes},
		},
		Blocks: map[string]schema.Block{
			"criteria": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"order":                   schema.Int64Attribute{Optional: true},
						"criteria":                schema.StringAttribute{Required: true},
						"operator":                schema.StringAttribute{Required: true},
						"value":                   schema.StringAttribute{Required: true},
						"and_or":                  schema.StringAttribute{Optional: true, Computed: true},
						"has_opening_parenthesis": schema.BoolAttribute{Optional: true},
						"has_closing_parenthesis": schema.BoolAttribute{Optional: true},
					},
				},
			},
		},
	}
}
