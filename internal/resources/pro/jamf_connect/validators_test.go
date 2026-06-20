// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package jamf_connect

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

var vRootType = tftypes.Object{AttributeTypes: map[string]tftypes.Type{
	"auto_deployment_type": tftypes.String,
	"version":              tftypes.String,
}}

func vSchema() schema.Schema {
	return schema.Schema{Attributes: map[string]schema.Attribute{
		"auto_deployment_type": schema.StringAttribute{Optional: true, Computed: true},
		"version":              schema.StringAttribute{Optional: true},
	}}
}

func strNull() tftypes.Value    { return tftypes.NewValue(tftypes.String, nil) }
func strUnknown() tftypes.Value { return tftypes.NewValue(tftypes.String, tftypes.UnknownValue) }
func strOf(s string) tftypes.Value {
	return tftypes.NewValue(tftypes.String, s)
}

func vCfg(deploymentType, version tftypes.Value) tfsdk.Config {
	return tfsdk.Config{
		Schema: vSchema(),
		Raw: tftypes.NewValue(vRootType, map[string]tftypes.Value{
			"auto_deployment_type": deploymentType,
			"version":              version,
		}),
	}
}

func runVersionValidator(c tfsdk.Config) bool {
	var resp resource.ValidateConfigResponse
	versionDeploymentTypeValidator{}.ValidateResource(
		context.Background(),
		resource.ValidateConfigRequest{Config: c},
		&resp,
	)
	for _, d := range resp.Diagnostics {
		if d.Severity() == diag.SeverityError {
			return true
		}
	}
	return false
}

func TestVersionDeploymentTypeValidator(t *testing.T) {
	cases := []struct {
		name       string
		deployment tftypes.Value
		version    tftypes.Value
		wantErr    bool
	}{
		{"none + no version ok", strOf(autoDeploymentNone), strNull(), false},
		{"null type (defaults NONE) + no version ok", strNull(), strNull(), false},
		{"none + version forbidden", strOf(autoDeploymentNone), strOf("2.45.1"), true},
		{"patch + version ok", strOf(autoDeploymentPatch), strOf("2.45.1"), false},
		{"patch + no version required", strOf(autoDeploymentPatch), strNull(), true},
		{"patch + empty version required", strOf(autoDeploymentPatch), strOf(""), true},
		{"initial-only + version ok", strOf(autoDeploymentInitialOnly), strOf("2.45.1"), false},
		{"unknown type defers", strUnknown(), strOf("2.45.1"), false},
		{"unknown version defers", strOf(autoDeploymentPatch), strUnknown(), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := runVersionValidator(vCfg(tc.deployment, tc.version)); got != tc.wantErr {
				t.Errorf("got err=%v, want err=%v", got, tc.wantErr)
			}
		})
	}
}
