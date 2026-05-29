// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package cloud_distribution_point

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestValidator_Descriptions(t *testing.T) {
	v := cdnTypeRequiredFieldsConfigValidator{}
	if v.Description(context.Background()) == "" {
		t.Error("Description must not be empty")
	}
	if v.MarkdownDescription(context.Background()) == "" {
		t.Error("MarkdownDescription must not be empty")
	}
}

// TestValidateCdnTypeRequiredFields exercises the pure validation logic across
// every cdn_type, including the unknown-deferral cases the STYLE_GUIDE requires
// (config validators must NOT error on unknown values sourced from variables /
// for_each / other resources).
func TestValidateCdnTypeRequiredFields(t *testing.T) {
	tests := []struct {
		name      string
		model     CloudDistributionPointResourceModel
		wantPaths []string // attribute paths expected to error (empty = no error)
	}{
		{
			name:  "cdn_type unknown defers",
			model: CloudDistributionPointResourceModel{CdnType: types.StringUnknown()},
		},
		{
			name:  "cdn_type null defers",
			model: CloudDistributionPointResourceModel{CdnType: types.StringNull()},
		},
		{
			name:  "JAMF_CLOUD needs nothing",
			model: CloudDistributionPointResourceModel{CdnType: types.StringValue("JAMF_CLOUD")},
		},
		{
			name: "RACKSPACE missing both",
			model: CloudDistributionPointResourceModel{
				CdnType:  types.StringValue("RACKSPACE_CLOUD_FILES"),
				Username: types.StringNull(),
				Password: types.StringNull(),
			},
			wantPaths: []string{"username", "password"},
		},
		{
			name: "RACKSPACE empty string counts as missing",
			model: CloudDistributionPointResourceModel{
				CdnType:  types.StringValue("RACKSPACE_CLOUD_FILES"),
				Username: types.StringValue(""),
				Password: types.StringValue("secret"),
			},
			wantPaths: []string{"username"},
		},
		{
			name: "AMAZON_S3 satisfied",
			model: CloudDistributionPointResourceModel{
				CdnType:  types.StringValue("AMAZON_S3"),
				Username: types.StringValue("AKIA..."),
				Password: types.StringValue("secret"),
			},
		},
		{
			name: "username unknown defers (AMAZON_S3)",
			model: CloudDistributionPointResourceModel{
				CdnType:  types.StringValue("AMAZON_S3"),
				Username: types.StringUnknown(),
				Password: types.StringValue("secret"),
			},
		},
		{
			name: "AKAMAI missing endpoint fields",
			model: CloudDistributionPointResourceModel{
				CdnType:     types.StringValue("AKAMAI"),
				Username:    types.StringValue("u"),
				Password:    types.StringValue("p"),
				UploadURL:   types.StringNull(),
				Directory:   types.StringValue("123"),
				DownloadURL: types.StringNull(),
			},
			wantPaths: []string{"upload_url", "download_url"},
		},
		{
			name: "AKAMAI fully satisfied",
			model: CloudDistributionPointResourceModel{
				CdnType:     types.StringValue("AKAMAI"),
				Username:    types.StringValue("u"),
				Password:    types.StringValue("p"),
				UploadURL:   types.StringValue("ftp://up"),
				Directory:   types.StringValue("123"),
				DownloadURL: types.StringValue("https://dl"),
			},
		},
		{
			name: "AMAZON_S3 signed URLs require private_key",
			model: CloudDistributionPointResourceModel{
				CdnType:           types.StringValue("AMAZON_S3"),
				Username:          types.StringValue("AKIA..."),
				Password:          types.StringValue("secret"),
				RequireSignedURLs: types.BoolValue(true),
				PrivateKey:        types.StringNull(),
			},
			wantPaths: []string{"private_key"},
		},
		{
			name: "AMAZON_S3 signed URLs with private_key ok",
			model: CloudDistributionPointResourceModel{
				CdnType:           types.StringValue("AMAZON_S3"),
				Username:          types.StringValue("AKIA..."),
				Password:          types.StringValue("secret"),
				RequireSignedURLs: types.BoolValue(true),
				PrivateKey:        types.StringValue("base64key"),
			},
		},
		{
			name: "AMAZON_S3 signed URLs but private_key unknown defers",
			model: CloudDistributionPointResourceModel{
				CdnType:           types.StringValue("AMAZON_S3"),
				Username:          types.StringValue("AKIA..."),
				Password:          types.StringValue("secret"),
				RequireSignedURLs: types.BoolValue(true),
				PrivateKey:        types.StringUnknown(),
			},
		},
		{
			name: "AMAZON_S3 require_signed_urls unknown defers private_key check",
			model: CloudDistributionPointResourceModel{
				CdnType:           types.StringValue("AMAZON_S3"),
				Username:          types.StringValue("AKIA..."),
				Password:          types.StringValue("secret"),
				RequireSignedURLs: types.BoolUnknown(),
				PrivateKey:        types.StringNull(),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			diags := validateCdnTypeRequiredFields(tc.model)
			if len(diags) != len(tc.wantPaths) {
				t.Fatalf("got %d diagnostics %v, want %d for paths %v", len(diags), diags, len(tc.wantPaths), tc.wantPaths)
			}
			got := map[string]bool{}
			for _, d := range diags {
				if dwp, ok := d.(diag.DiagnosticWithPath); ok {
					got[dwp.Path().String()] = true
				}
			}
			for _, want := range tc.wantPaths {
				if !got[want] {
					t.Errorf("expected error on attribute %q; diags=%v", want, diags)
				}
			}
		})
	}
}
