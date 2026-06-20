// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package file_share_distribution_point

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestValidator_Descriptions(t *testing.T) {
	for _, v := range []interface {
		Description(context.Context) string
		MarkdownDescription(context.Context) string
	}{transportConfigValidator{}, loadBalancingConfigValidator{}} {
		if v.Description(context.Background()) == "" {
			t.Error("Description must not be empty")
		}
		if v.MarkdownDescription(context.Background()) == "" {
			t.Error("MarkdownDescription must not be empty")
		}
	}
}

func TestValidateTransport(t *testing.T) {
	tests := []struct {
		name      string
		model     FileShareDistributionPointResourceModel
		wantError bool
	}{
		{
			name:  "connection type unknown defers",
			model: FileShareDistributionPointResourceModel{FileSharingConnectionType: types.StringUnknown()},
		},
		{
			name:  "connection type null defers",
			model: FileShareDistributionPointResourceModel{FileSharingConnectionType: types.StringNull()},
		},
		{
			name:  "SMB satisfies transport regardless of https",
			model: FileShareDistributionPointResourceModel{FileSharingConnectionType: types.StringValue(connectionTypeSMB), HTTPSEnabled: types.BoolValue(false)},
		},
		{
			name:  "AFP satisfies transport",
			model: FileShareDistributionPointResourceModel{FileSharingConnectionType: types.StringValue(connectionTypeAFP)},
		},
		{
			name:  "NONE with https enabled is fine",
			model: FileShareDistributionPointResourceModel{FileSharingConnectionType: types.StringValue(connectionTypeNone), HTTPSEnabled: types.BoolValue(true)},
		},
		{
			name:  "NONE with https omitted defers (may be preserved)",
			model: FileShareDistributionPointResourceModel{FileSharingConnectionType: types.StringValue(connectionTypeNone), HTTPSEnabled: types.BoolNull()},
		},
		{
			name:  "NONE with https unknown defers",
			model: FileShareDistributionPointResourceModel{FileSharingConnectionType: types.StringValue(connectionTypeNone), HTTPSEnabled: types.BoolUnknown()},
		},
		{
			name:      "NONE with https explicitly false errors",
			model:     FileShareDistributionPointResourceModel{FileSharingConnectionType: types.StringValue(connectionTypeNone), HTTPSEnabled: types.BoolValue(false)},
			wantError: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			diags := validateTransport(tc.model)
			if diags.HasError() != tc.wantError {
				t.Errorf("validateTransport hasError=%v, want %v (%v)", diags.HasError(), tc.wantError, diags)
			}
		})
	}
}

func TestValidateLoadBalancing(t *testing.T) {
	tests := []struct {
		name      string
		model     FileShareDistributionPointResourceModel
		wantError bool
	}{
		{
			name:  "load balancing unknown defers",
			model: FileShareDistributionPointResourceModel{EnableLoadBalancing: types.BoolUnknown()},
		},
		{
			name:  "load balancing null defers",
			model: FileShareDistributionPointResourceModel{EnableLoadBalancing: types.BoolNull()},
		},
		{
			name:  "load balancing false is fine",
			model: FileShareDistributionPointResourceModel{EnableLoadBalancing: types.BoolValue(false), BackupDistributionPointID: types.StringValue(noneBackupSentinel)},
		},
		{
			name:  "load balancing true with backup omitted defers",
			model: FileShareDistributionPointResourceModel{EnableLoadBalancing: types.BoolValue(true), BackupDistributionPointID: types.StringNull()},
		},
		{
			name:  "load balancing true with real DP id is fine",
			model: FileShareDistributionPointResourceModel{EnableLoadBalancing: types.BoolValue(true), BackupDistributionPointID: types.StringValue("42")},
		},
		{
			name:      "load balancing true with none sentinel errors",
			model:     FileShareDistributionPointResourceModel{EnableLoadBalancing: types.BoolValue(true), BackupDistributionPointID: types.StringValue(noneBackupSentinel)},
			wantError: true,
		},
		{
			name:      "load balancing true with cloud sentinel errors",
			model:     FileShareDistributionPointResourceModel{EnableLoadBalancing: types.BoolValue(true), BackupDistributionPointID: types.StringValue(cloudBackupSentinel)},
			wantError: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			diags := validateLoadBalancing(tc.model)
			if diags.HasError() != tc.wantError {
				t.Errorf("validateLoadBalancing hasError=%v, want %v (%v)", diags.HasError(), tc.wantError, diags)
			}
		})
	}
}
