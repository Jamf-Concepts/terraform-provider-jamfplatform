// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package icon

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

// iconTestModel builds an icon model that survives a schema round-trip.
func iconTestModel(id, source, hash, url string) IconResourceModel {
	optional := func(v string) types.String {
		if v == "" {
			return types.StringNull()
		}
		return types.StringValue(v)
	}
	return IconResourceModel{
		ID:             optional(id),
		IconFileSource: optional(source),
		SourceHash:     optional(hash),
		URL:            optional(url),
		Timeouts:       helpers.NewResourceTimeoutsNullValue(iconTimeoutAttributeTypes),
	}
}

// runIconModifyPlan drives ModifyPlan the way the framework does, with resp.Plan
// pre-populated from req.Plan. A nil stateModel is a create, and a nil planModel
// is a destroy.
func runIconModifyPlan(t *testing.T, stateModel, planModel *IconResourceModel) *resource.ModifyPlanResponse {
	t.Helper()
	ctx := context.Background()

	var schemaResp resource.SchemaResponse
	NewIconResource().(*IconResource).Schema(ctx, resource.SchemaRequest{}, &schemaResp)
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

	r := &IconResource{}
	resp := &resource.ModifyPlanResponse{Plan: plan}
	r.ModifyPlan(ctx, resource.ModifyPlanRequest{Plan: plan, State: state}, resp)
	return resp
}

// plannedIcon decodes the model ModifyPlan left on the response.
func plannedIcon(t *testing.T, resp *resource.ModifyPlanResponse) IconResourceModel {
	t.Helper()
	var planned IconResourceModel
	if diags := resp.Plan.Get(context.Background(), &planned); diags.HasError() {
		t.Fatalf("planned get: %v", diags)
	}
	return planned
}

// requiresReplaceOn reports whether the response asks Terraform to replace the
// resource because of the named attribute.
func requiresReplaceOn(resp *resource.ModifyPlanResponse, attribute string) bool {
	for _, p := range resp.RequiresReplace {
		if p.Equal(path.Root(attribute)) {
			return true
		}
	}
	return false
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

// TestModifyPlanCreateLeavesSourceHashUnknown is the regression guard for issue
// #373. A create must not plan a hash: the value is resolved in Create from the
// bytes it uploads, so a source that answers two reads with different bytes
// cannot plan one value and apply another.
func TestModifyPlanCreateLeavesSourceHashUnknown(t *testing.T) {
	url, hits := unstableIconServer(t)
	planModel := iconTestModel("", url, "", "")
	planModel.SourceHash = types.StringUnknown()
	planModel.ID = types.StringUnknown()
	planModel.URL = types.StringUnknown()

	resp := runIconModifyPlan(t, nil, &planModel)
	if resp.Diagnostics.HasError() {
		t.Fatalf("ModifyPlan diags: %v", resp.Diagnostics)
	}

	if got := plannedIcon(t, resp); !got.SourceHash.IsUnknown() {
		t.Fatalf("source_hash was planned as %v; a create must leave it unknown for Create to resolve", got.SourceHash)
	}
	if n := hits.Load(); n != 0 {
		t.Fatalf("the source was fetched %d times during a create plan; it must not be read before apply", n)
	}
	if len(resp.RequiresReplace) != 0 {
		t.Fatalf("a create asked for a replacement: %v", resp.RequiresReplace)
	}
}

// TestModifyPlanCreateFromLocalPathLeavesSourceHashUnknown pins that the create
// rule is uniform. A local file's bytes are stable and could be hashed at plan
// time, but Create hashes what it uploads for every source, so one rule covers
// both rather than two code paths agreeing by accident.
func TestModifyPlanCreateFromLocalPathLeavesSourceHashUnknown(t *testing.T) {
	planModel := iconTestModel("", writeIconFile(t, "icon.png", "local bytes"), "", "")
	planModel.SourceHash = types.StringUnknown()

	resp := runIconModifyPlan(t, nil, &planModel)
	if resp.Diagnostics.HasError() {
		t.Fatalf("ModifyPlan diags: %v", resp.Diagnostics)
	}
	if got := plannedIcon(t, resp); !got.SourceHash.IsUnknown() {
		t.Fatalf("source_hash was planned as %v; a create must leave it unknown", got.SourceHash)
	}
}

// TestModifyPlanCreateFromMissingLocalPathErrors pins that the create still
// probes the source. The hash it computes is discarded, but opening the file is
// what turns a typo in a path into a plan-time failure rather than one part-way
// through an apply.
func TestModifyPlanCreateFromMissingLocalPathErrors(t *testing.T) {
	planModel := iconTestModel("", filepath.Join(t.TempDir(), "typo.png"), "", "")
	planModel.ID = types.StringUnknown()
	planModel.SourceHash = types.StringUnknown()
	planModel.URL = types.StringUnknown()

	resp := runIconModifyPlan(t, nil, &planModel)
	if !resp.Diagnostics.HasError() {
		t.Fatal("a create from a missing local file planned without a diagnostic")
	}
}

// TestModifyPlanLocalSourceUnchangedContentIsNoOp covers a local path whose
// bytes still hash to what state holds.
func TestModifyPlanLocalSourceUnchangedContentIsNoOp(t *testing.T) {
	source := writeIconFile(t, "icon.png", "local bytes")
	hash := files.ComputeContentSHA256([]byte("local bytes"))

	stateModel := iconTestModel("42", source, hash, "https://cdn.example.com/icon")
	planModel := iconTestModel("42", source, hash, "https://cdn.example.com/icon")

	resp := runIconModifyPlan(t, &stateModel, &planModel)
	if resp.Diagnostics.HasError() {
		t.Fatalf("ModifyPlan diags: %v", resp.Diagnostics)
	}
	if len(resp.RequiresReplace) != 0 {
		t.Fatalf("unchanged local content asked for a replacement: %v", resp.RequiresReplace)
	}
	if got := plannedIcon(t, resp); got.SourceHash.ValueString() != hash {
		t.Fatalf("source_hash = %q, want the stored %q", got.SourceHash.ValueString(), hash)
	}
}

// TestModifyPlanLocalSourceChangedContentReplaces covers a local path edited in
// place. The hash is what changed, so it is the attribute Terraform reports as
// forcing the replacement, and it goes unknown so Create resolves it.
func TestModifyPlanLocalSourceChangedContentReplaces(t *testing.T) {
	source := writeIconFile(t, "icon.png", "new bytes")
	stored := files.ComputeContentSHA256([]byte("old bytes"))

	stateModel := iconTestModel("42", source, stored, "https://cdn.example.com/icon")
	planModel := iconTestModel("42", source, stored, "https://cdn.example.com/icon")

	resp := runIconModifyPlan(t, &stateModel, &planModel)
	if resp.Diagnostics.HasError() {
		t.Fatalf("ModifyPlan diags: %v", resp.Diagnostics)
	}
	if !requiresReplaceOn(resp, "source_hash") {
		t.Fatalf("changed local content did not force a replacement on source_hash: %v", resp.RequiresReplace)
	}

	got := plannedIcon(t, resp)
	if !got.SourceHash.IsUnknown() {
		t.Fatalf("source_hash = %v, want unknown so Create resolves it from the uploaded bytes", got.SourceHash)
	}
	if !got.ID.IsUnknown() || !got.URL.IsUnknown() {
		t.Fatalf("id = %v and url = %v, want both unknown on a replacement", got.ID, got.URL)
	}
}

// TestModifyPlanLocalSourceMissingFileErrors keeps the plan-time failure for an
// unreadable local path, which is where an operator can still fix it.
func TestModifyPlanLocalSourceMissingFileErrors(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "absent.png")
	stateModel := iconTestModel("42", missing, "sha256:whatever", "https://cdn.example.com/icon")
	planModel := iconTestModel("42", missing, "sha256:whatever", "https://cdn.example.com/icon")

	resp := runIconModifyPlan(t, &stateModel, &planModel)
	if !resp.Diagnostics.HasError() {
		t.Fatal("a missing local icon file planned without a diagnostic")
	}
}

// TestModifyPlanURLSourceUnchangedIsNoOpWithoutFetching is the quieter half of
// issue #373: an unstable URL hashed on every plan proposes a replacement
// nothing asked for. An unchanged URL string must plan nothing and read nothing.
func TestModifyPlanURLSourceUnchangedIsNoOpWithoutFetching(t *testing.T) {
	url, hits := unstableIconServer(t)
	hash := files.ComputeContentSHA256([]byte("whatever the CDN served last time"))

	stateModel := iconTestModel("42", url, hash, "https://cdn.example.com/icon")
	planModel := iconTestModel("42", url, hash, "https://cdn.example.com/icon")

	resp := runIconModifyPlan(t, &stateModel, &planModel)
	if resp.Diagnostics.HasError() {
		t.Fatalf("ModifyPlan diags: %v", resp.Diagnostics)
	}
	if n := hits.Load(); n != 0 {
		t.Fatalf("the URL was fetched %d times during a plan; remote content is not stably observable", n)
	}
	if len(resp.RequiresReplace) != 0 {
		t.Fatalf("an unchanged URL asked for a replacement: %v", resp.RequiresReplace)
	}
	if got := plannedIcon(t, resp); got.SourceHash.ValueString() != hash {
		t.Fatalf("source_hash = %q, want the stored %q", got.SourceHash.ValueString(), hash)
	}
}

// TestModifyPlanURLSourceChangedReplacesWithoutFetching covers re-pointing the
// URL. The string is what changed, so icon_file_source is the reported trigger.
func TestModifyPlanURLSourceChangedReplacesWithoutFetching(t *testing.T) {
	url, hits := unstableIconServer(t)
	hash := files.ComputeContentSHA256([]byte("bytes from the old URL"))

	stateModel := iconTestModel("42", url, hash, "https://cdn.example.com/icon")
	planModel := iconTestModel("42", url+"?v=2", hash, "https://cdn.example.com/icon")

	resp := runIconModifyPlan(t, &stateModel, &planModel)
	if resp.Diagnostics.HasError() {
		t.Fatalf("ModifyPlan diags: %v", resp.Diagnostics)
	}
	if n := hits.Load(); n != 0 {
		t.Fatalf("the URL was fetched %d times during a plan; the string is what is compared", n)
	}
	if !requiresReplaceOn(resp, "icon_file_source") {
		t.Fatalf("a changed URL did not force a replacement on icon_file_source: %v", resp.RequiresReplace)
	}
	if got := plannedIcon(t, resp); !got.SourceHash.IsUnknown() {
		t.Fatalf("source_hash = %v, want unknown on a replacement", got.SourceHash)
	}
}

// TestModifyPlanURLSourceAdoptedAfterImportReplaces records the cost of
// comparing URLs by string. Import leaves icon_file_source null, so any URL
// assigned afterwards reads as a change. Hashing would not save it either: Jamf
// Pro re-encodes an uploaded PNG, so the CDN bytes import hashed never match the
// upstream ones.
func TestModifyPlanURLSourceAdoptedAfterImportReplaces(t *testing.T) {
	url, _ := unstableIconServer(t)
	hash := files.ComputeContentSHA256([]byte("bytes Jamf Pro served back"))

	stateModel := iconTestModel("42", "", hash, "https://cdn.example.com/icon")
	planModel := iconTestModel("42", url, hash, "https://cdn.example.com/icon")

	resp := runIconModifyPlan(t, &stateModel, &planModel)
	if resp.Diagnostics.HasError() {
		t.Fatalf("ModifyPlan diags: %v", resp.Diagnostics)
	}
	if !requiresReplaceOn(resp, "icon_file_source") {
		t.Fatalf("adopting a URL after import did not force a replacement: %v", resp.RequiresReplace)
	}
}

// TestModifyPlanUnknownSourceKeepsStoredHash covers a source interpolated from
// another resource, which is unknown until that resource applies. The stored
// hash must survive so the plan stays consistent with what apply commits.
func TestModifyPlanUnknownSourceKeepsStoredHash(t *testing.T) {
	hash := files.ComputeContentSHA256([]byte("stored bytes"))
	stateModel := iconTestModel("42", "./icon.png", hash, "https://cdn.example.com/icon")
	planModel := iconTestModel("42", "./icon.png", hash, "https://cdn.example.com/icon")
	planModel.IconFileSource = types.StringUnknown()

	resp := runIconModifyPlan(t, &stateModel, &planModel)
	if resp.Diagnostics.HasError() {
		t.Fatalf("ModifyPlan diags: %v", resp.Diagnostics)
	}
	if len(resp.RequiresReplace) != 0 {
		t.Fatalf("an unknown source asked for a replacement: %v", resp.RequiresReplace)
	}
	if got := plannedIcon(t, resp); got.SourceHash.ValueString() != hash {
		t.Fatalf("source_hash = %q, want the stored %q", got.SourceHash.ValueString(), hash)
	}
}

// TestModifyPlanDestroyReadsNothing pins the destroy early return. A destroy has
// no plan to modify and no reason to reach a file or a URL.
func TestModifyPlanDestroyReadsNothing(t *testing.T) {
	url, hits := unstableIconServer(t)
	stateModel := iconTestModel("42", url, files.ComputeContentSHA256([]byte("stored")), "https://cdn.example.com/icon")

	resp := runIconModifyPlan(t, &stateModel, nil)
	if resp.Diagnostics.HasError() {
		t.Fatalf("ModifyPlan diags on destroy: %v", resp.Diagnostics)
	}
	if n := hits.Load(); n != 0 {
		t.Fatalf("the URL was fetched %d times during a destroy plan", n)
	}
	if len(resp.RequiresReplace) != 0 {
		t.Fatalf("a destroy asked for a replacement: %v", resp.RequiresReplace)
	}
}
