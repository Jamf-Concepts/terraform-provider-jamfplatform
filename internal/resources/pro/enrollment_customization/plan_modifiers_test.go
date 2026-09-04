// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package enrollment_customization

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/files"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// iconPlanTestModel builds a customization model that survives a schema
// round-trip, with only the icon-bearing fields varying.
func iconPlanTestModel(id, iconSource, iconHash, iconURL string) EnrollmentCustomizationResourceModel {
	optional := func(v string) types.String {
		if v == "" {
			return types.StringNull()
		}
		return types.StringValue(v)
	}
	return EnrollmentCustomizationResourceModel{
		ID:             optional(id),
		DisplayName:    types.StringValue("tf-acc-icon-plan"),
		Description:    types.StringValue("icon plan modifier coverage"),
		SiteID:         types.StringValue("-1"),
		IconSource:     optional(iconSource),
		IconSourceHash: optional(iconHash),
		BrandingSettings: &brandingSettingsModel{
			BodyTextColor:   types.StringValue("333333"),
			ButtonColor:     types.StringValue("0066cc"),
			ButtonTextColor: types.StringValue("ffffff"),
			BackgroundColor: types.StringValue("ffffff"),
			IconURL:         optional(iconURL),
		},
		Timeouts: helpers.NewResourceTimeoutsNullValue(enrollmentCustomizationTimeoutAttributeTypes),
	}
}

// runIconModifyPlan drives ModifyPlan the way the framework does, with resp.Plan
// pre-populated from req.Plan. A nil stateModel is a create, and a nil planModel
// is a destroy.
func runIconModifyPlan(t *testing.T, stateModel, planModel *EnrollmentCustomizationResourceModel) *resource.ModifyPlanResponse {
	t.Helper()
	ctx := context.Background()

	var schemaResp resource.SchemaResponse
	NewEnrollmentCustomizationResource().(*EnrollmentCustomizationResource).Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema diags: %v", schemaResp.Diagnostics)
	}
	nullRaw := tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil)

	state := tfsdk.State{Schema: schemaResp.Schema, Raw: nullRaw}
	if stateModel != nil {
		if diags := state.Set(ctx, stateModel); diags.HasError() {
			t.Fatalf("state set: %v", diags)
		}
	}
	plan := tfsdk.Plan{Schema: schemaResp.Schema, Raw: nullRaw}
	if planModel != nil {
		if diags := plan.Set(ctx, planModel); diags.HasError() {
			t.Fatalf("plan set: %v", diags)
		}
	}

	r := &EnrollmentCustomizationResource{}
	resp := &resource.ModifyPlanResponse{Plan: plan}
	r.ModifyPlan(ctx, resource.ModifyPlanRequest{Plan: plan, State: state}, resp)
	return resp
}

// plannedIcon reads the two values an icon upload settles off the response.
func plannedIcon(t *testing.T, resp *resource.ModifyPlanResponse) (types.String, types.String) {
	t.Helper()
	ctx := context.Background()

	var hash types.String
	if diags := resp.Plan.GetAttribute(ctx, path.Root("icon_source_hash"), &hash); diags.HasError() {
		t.Fatalf("reading planned icon_source_hash: %v", diags)
	}
	var url types.String
	if diags := resp.Plan.GetAttribute(ctx, path.Root("branding_settings").AtName("icon_url"), &url); diags.HasError() {
		t.Fatalf("reading planned icon_url: %v", diags)
	}
	return hash, url
}

// writeIconFile writes bytes to a file in a per-test directory and returns its
// path.
func writeIconFile(t *testing.T, name, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatalf("writing %s: %v", p, err)
	}
	return p
}

// unstableIconServer serves a different body on every request, the shape Apple's
// iTunes artwork CDN has for a fixed URL. The returned counter records how many
// times the provider fetched it.
func unstableIconServer(t *testing.T) (string, *atomic.Int64) {
	t.Helper()
	var hits atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := hits.Add(1)
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte(strings.Repeat("x", int(n))))
	}))
	t.Cleanup(server.Close)
	return server.URL + "/icon.png", &hits
}

// TestModifyPlanCreateLeavesIconSourceHashUnknown is the regression guard for
// issue #373 on this resource, where it was reproduced on a live tenant on
// 2026-09-04. A create must not plan a hash: Create hashes the exact bytes it
// uploads.
func TestModifyPlanCreateLeavesIconSourceHashUnknown(t *testing.T) {
	url, hits := unstableIconServer(t)
	planModel := iconPlanTestModel("", url, "", "")
	planModel.ID = types.StringUnknown()
	planModel.IconSourceHash = types.StringUnknown()
	planModel.BrandingSettings.IconURL = types.StringUnknown()

	resp := runIconModifyPlan(t, nil, &planModel)
	if resp.Diagnostics.HasError() {
		t.Fatalf("ModifyPlan diags: %v", resp.Diagnostics)
	}

	hash, _ := plannedIcon(t, resp)
	if !hash.IsUnknown() {
		t.Fatalf("icon_source_hash was planned as %v; a create must leave it unknown for Create to resolve", hash)
	}
	if n := hits.Load(); n != 0 {
		t.Fatalf("the source was fetched %d times during a create plan; it must not be read before apply", n)
	}
}

// TestModifyPlanCreateFromLocalPathLeavesIconSourceHashUnknown pins that the
// create rule is uniform. A local file's bytes are stable and could be hashed at
// plan time, but Create hashes what it uploads for every source, so one rule
// covers both rather than two code paths agreeing by accident.
func TestModifyPlanCreateFromLocalPathLeavesIconSourceHashUnknown(t *testing.T) {
	planModel := iconPlanTestModel("", writeIconFile(t, "icon.png", "local bytes"), "", "")
	planModel.ID = types.StringUnknown()
	planModel.IconSourceHash = types.StringUnknown()
	planModel.BrandingSettings.IconURL = types.StringUnknown()

	resp := runIconModifyPlan(t, nil, &planModel)
	if resp.Diagnostics.HasError() {
		t.Fatalf("ModifyPlan diags: %v", resp.Diagnostics)
	}
	if hash, _ := plannedIcon(t, resp); !hash.IsUnknown() {
		t.Fatalf("icon_source_hash was planned as %v; a create must leave it unknown", hash)
	}
}

// TestModifyPlanLocalIconUnchangedPlansNoUpload covers a local path whose bytes
// still hash to what state holds. Both computed values must stay known, or
// Update re-uploads on every apply that touches anything else.
func TestModifyPlanLocalIconUnchangedPlansNoUpload(t *testing.T) {
	source := writeIconFile(t, "icon.png", "local bytes")
	stored := files.ComputeContentSHA256([]byte("local bytes"))
	const iconURL = "https://tenant.example/api/v2/enrollment-customizations/images/3"

	stateModel := iconPlanTestModel("7", source, stored, iconURL)
	planModel := iconPlanTestModel("7", source, stored, iconURL)

	resp := runIconModifyPlan(t, &stateModel, &planModel)
	if resp.Diagnostics.HasError() {
		t.Fatalf("ModifyPlan diags: %v", resp.Diagnostics)
	}

	hash, url := plannedIcon(t, resp)
	if hash.ValueString() != stored {
		t.Fatalf("icon_source_hash = %v, want the stored %q; an unknown hash makes Update re-upload", hash, stored)
	}
	if url.ValueString() != iconURL {
		t.Fatalf("icon_url = %v, want the stored %q", url, iconURL)
	}
}

// TestModifyPlanLocalIconChangedPlansUpload covers a local path edited in place.
// Both values the upload settles go unknown together: the hash is the signal
// Update reads, and icon_url is what the upload returns.
func TestModifyPlanLocalIconChangedPlansUpload(t *testing.T) {
	source := writeIconFile(t, "icon.png", "new bytes")
	stored := files.ComputeContentSHA256([]byte("old bytes"))
	const iconURL = "https://tenant.example/api/v2/enrollment-customizations/images/3"

	stateModel := iconPlanTestModel("7", source, stored, iconURL)
	planModel := iconPlanTestModel("7", source, stored, iconURL)

	resp := runIconModifyPlan(t, &stateModel, &planModel)
	if resp.Diagnostics.HasError() {
		t.Fatalf("ModifyPlan diags: %v", resp.Diagnostics)
	}

	hash, url := plannedIcon(t, resp)
	if !hash.IsUnknown() {
		t.Fatalf("icon_source_hash = %v, want unknown so Update uploads and commits the hash of what it sent", hash)
	}
	if !url.IsUnknown() {
		t.Fatalf("icon_url = %v, want unknown so apply can supply the new upload URL", url)
	}
	if len(resp.RequiresReplace) != 0 {
		t.Fatalf("a customization has an update endpoint, so an icon change must not replace it: %v", resp.RequiresReplace)
	}
}

// TestModifyPlanLocalIconMissingFileErrors keeps the plan-time failure for an
// unreadable local path, which is where an operator can still fix it.
func TestModifyPlanLocalIconMissingFileErrors(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "absent.png")
	stateModel := iconPlanTestModel("7", missing, "sha256:whatever", "https://tenant.example/images/3")
	planModel := iconPlanTestModel("7", missing, "sha256:whatever", "https://tenant.example/images/3")

	resp := runIconModifyPlan(t, &stateModel, &planModel)
	if !resp.Diagnostics.HasError() {
		t.Fatal("a missing local icon file planned without a diagnostic")
	}
}

// TestModifyPlanURLIconUnchangedPlansNoUploadWithoutFetching is the quieter half
// of issue #373: an unstable URL hashed on every plan proposes an upload nothing
// asked for. An unchanged URL string must plan nothing and read nothing.
func TestModifyPlanURLIconUnchangedPlansNoUploadWithoutFetching(t *testing.T) {
	url, hits := unstableIconServer(t)
	stored := files.ComputeContentSHA256([]byte("whatever the CDN served last time"))
	const iconURL = "https://tenant.example/api/v2/enrollment-customizations/images/3"

	stateModel := iconPlanTestModel("7", url, stored, iconURL)
	planModel := iconPlanTestModel("7", url, stored, iconURL)

	resp := runIconModifyPlan(t, &stateModel, &planModel)
	if resp.Diagnostics.HasError() {
		t.Fatalf("ModifyPlan diags: %v", resp.Diagnostics)
	}
	if n := hits.Load(); n != 0 {
		t.Fatalf("the URL was fetched %d times during a plan; remote content is not stably observable", n)
	}
	if hash, _ := plannedIcon(t, resp); hash.ValueString() != stored {
		t.Fatalf("icon_source_hash = %v, want the stored %q", hash, stored)
	}
}

// TestModifyPlanURLIconChangedPlansUploadWithoutFetching covers re-pointing the
// URL, which is the only URL change this resource can detect.
func TestModifyPlanURLIconChangedPlansUploadWithoutFetching(t *testing.T) {
	url, hits := unstableIconServer(t)
	stored := files.ComputeContentSHA256([]byte("bytes from the old URL"))
	const iconURL = "https://tenant.example/api/v2/enrollment-customizations/images/3"

	stateModel := iconPlanTestModel("7", url, stored, iconURL)
	planModel := iconPlanTestModel("7", url+"?v=2", stored, iconURL)

	resp := runIconModifyPlan(t, &stateModel, &planModel)
	if resp.Diagnostics.HasError() {
		t.Fatalf("ModifyPlan diags: %v", resp.Diagnostics)
	}
	if n := hits.Load(); n != 0 {
		t.Fatalf("the URL was fetched %d times during a plan; the string is what is compared", n)
	}

	hash, plannedURL := plannedIcon(t, resp)
	if !hash.IsUnknown() {
		t.Fatalf("icon_source_hash = %v, want unknown so Update uploads from the new URL", hash)
	}
	if !plannedURL.IsUnknown() {
		t.Fatalf("icon_url = %v, want unknown so apply can supply the new upload URL", plannedURL)
	}
}

// TestModifyPlanNoIconSourcePlansNoUpload covers a customization whose icon is
// managed out of band through branding_settings.icon_url. Nothing may be read
// and nothing may go unknown, or Update would upload a source that is not set.
func TestModifyPlanNoIconSourcePlansNoUpload(t *testing.T) {
	const iconURL = "https://tenant.example/api/v2/enrollment-customizations/images/3"
	stateModel := iconPlanTestModel("7", "", "", iconURL)
	planModel := iconPlanTestModel("7", "", "", iconURL)

	resp := runIconModifyPlan(t, &stateModel, &planModel)
	if resp.Diagnostics.HasError() {
		t.Fatalf("ModifyPlan diags: %v", resp.Diagnostics)
	}

	hash, url := plannedIcon(t, resp)
	if hash.IsUnknown() {
		t.Fatal("icon_source_hash went unknown with no icon_source set, which would make Update upload nothing and commit an unknown")
	}
	if url.ValueString() != iconURL {
		t.Fatalf("icon_url = %v, want the configured %q left alone", url, iconURL)
	}
}

// TestModifyPlanUnknownIconSourceKeepsStoredHash covers a source interpolated
// from another resource, which is unknown until that resource applies.
func TestModifyPlanUnknownIconSourceKeepsStoredHash(t *testing.T) {
	stored := files.ComputeContentSHA256([]byte("stored bytes"))
	const iconURL = "https://tenant.example/api/v2/enrollment-customizations/images/3"

	stateModel := iconPlanTestModel("7", "./icon.png", stored, iconURL)
	planModel := iconPlanTestModel("7", "./icon.png", stored, iconURL)
	planModel.IconSource = types.StringUnknown()

	resp := runIconModifyPlan(t, &stateModel, &planModel)
	if resp.Diagnostics.HasError() {
		t.Fatalf("ModifyPlan diags: %v", resp.Diagnostics)
	}
	if hash, _ := plannedIcon(t, resp); hash.ValueString() != stored {
		t.Fatalf("icon_source_hash = %v, want the stored %q", hash, stored)
	}
}

// TestModifyPlanDestroyReadsNothing pins the destroy early return. A destroy has
// no plan to modify and no reason to reach a file or a URL.
func TestModifyPlanDestroyReadsNothing(t *testing.T) {
	url, hits := unstableIconServer(t)
	stateModel := iconPlanTestModel("7", url, files.ComputeContentSHA256([]byte("stored")), "https://tenant.example/images/3")

	resp := runIconModifyPlan(t, &stateModel, nil)
	if resp.Diagnostics.HasError() {
		t.Fatalf("ModifyPlan diags on destroy: %v", resp.Diagnostics)
	}
	if n := hits.Load(); n != 0 {
		t.Fatalf("the URL was fetched %d times during a destroy plan", n)
	}
}
