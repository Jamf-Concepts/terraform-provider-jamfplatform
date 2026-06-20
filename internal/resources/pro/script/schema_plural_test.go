// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package script

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

func TestScriptsDataSource_Metadata(t *testing.T) {
	d := NewScriptsDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*ScriptsDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_scripts" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_scripts", resp.TypeName)
	}
}

func TestScriptsDataSource_Schema(t *testing.T) {
	d := NewScriptsDataSource()
	var resp datasource.SchemaResponse
	d.(*ScriptsDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	for _, name := range []string{"id", "scripts", "filter", "timeouts"} {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
}
