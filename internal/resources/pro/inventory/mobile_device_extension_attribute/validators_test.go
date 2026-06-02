// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mobile_device_extension_attribute

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func configFromAttrs(t *testing.T, attrs map[string]tftypes.Value) tfsdk.Config {
	t.Helper()
	r := NewMobileDeviceExtensionAttributeResource()
	var sr resource.SchemaResponse
	r.(*MobileDeviceExtensionAttributeResource).Schema(context.Background(), resource.SchemaRequest{}, &sr)
	if sr.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", sr.Diagnostics)
	}
	objType := sr.Schema.Type().TerraformType(context.Background()).(tftypes.Object)
	values := map[string]tftypes.Value{}
	for name, typ := range objType.AttributeTypes {
		if v, ok := attrs[name]; ok {
			values[name] = v
			continue
		}
		values[name] = tftypes.NewValue(typ, nil)
	}
	return tfsdk.Config{Schema: sr.Schema, Raw: tftypes.NewValue(objType, values)}
}

func str(s string) tftypes.Value { return tftypes.NewValue(tftypes.String, s) }
func boolv(b bool) tftypes.Value { return tftypes.NewValue(tftypes.Bool, b) }
func popup(vs ...string) tftypes.Value {
	elems := make([]tftypes.Value, len(vs))
	for i, v := range vs {
		elems[i] = tftypes.NewValue(tftypes.String, v)
	}
	return tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, elems)
}

func TestInputTypeValidator_Mobile(t *testing.T) {
	tests := []struct {
		name    string
		attrs   map[string]tftypes.Value
		wantErr bool
	}{
		{"text bare", map[string]tftypes.Value{"input_type": str("TEXT")}, false},
		{"popup with choices", map[string]tftypes.Value{"input_type": str("POPUP"), "popup_menu_choices": popup("a")}, false},
		{"popup no choices ok", map[string]tftypes.Value{"input_type": str("POPUP")}, false},
		{"dsam with attribute", map[string]tftypes.Value{"input_type": str("DIRECTORY_SERVICE_ATTRIBUTE_MAPPING"), "directory_service_attribute": str("mail")}, false},

		{"dsam missing attribute", map[string]tftypes.Value{"input_type": str("DIRECTORY_SERVICE_ATTRIBUTE_MAPPING")}, true},
		{"dsam empty attribute", map[string]tftypes.Value{"input_type": str("DIRECTORY_SERVICE_ATTRIBUTE_MAPPING"), "directory_service_attribute": str("")}, true},
		{"text with popup", map[string]tftypes.Value{"input_type": str("TEXT"), "popup_menu_choices": popup("a")}, true},
		{"text with dsa", map[string]tftypes.Value{"input_type": str("TEXT"), "directory_service_attribute": str("mail")}, true},
		{"popup with dsa", map[string]tftypes.Value{"input_type": str("POPUP"), "popup_menu_choices": popup("a"), "directory_service_attribute": str("mail")}, true},
		{"dsam with popup", map[string]tftypes.Value{"input_type": str("DIRECTORY_SERVICE_ATTRIBUTE_MAPPING"), "directory_service_attribute": str("mail"), "popup_menu_choices": popup("a")}, true},
		{"text allow multiple forbidden", map[string]tftypes.Value{"input_type": str("TEXT"), "allow_multiple_values": boolv(true)}, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := resource.ValidateConfigRequest{Config: configFromAttrs(t, tc.attrs)}
			var resp resource.ValidateConfigResponse
			inputTypeConfigValidator{}.ValidateResource(context.Background(), req, &resp)
			if got := resp.Diagnostics.HasError(); got != tc.wantErr {
				t.Errorf("wantErr=%v, got=%v (diags: %v)", tc.wantErr, got, resp.Diagnostics)
			}
		})
	}
}
