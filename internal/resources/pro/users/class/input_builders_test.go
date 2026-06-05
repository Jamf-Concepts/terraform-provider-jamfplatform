// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package class

import (
	"context"
	"encoding/xml"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// strSet builds a Terraform set from the given values. With no values it returns
// a non-null EMPTY set (the variadic would otherwise be a nil slice, which
// SetValueFrom turns into a null set).
func strSet(t *testing.T, vals ...string) types.Set {
	t.Helper()
	if vals == nil {
		vals = []string{}
	}
	s, d := types.SetValueFrom(context.Background(), types.StringType, vals)
	if d.HasError() {
		t.Fatalf("set value: %v", d)
	}
	return s
}

func TestBuildInput_AlwaysEmitsMembershipWrappers(t *testing.T) {
	// Null collections must still serialise as non-nil empty wrappers so the
	// classic PUT clears them instead of merging (leaving server values intact).
	plan := ClassResourceModel{
		Name:                 types.StringValue("empty class"),
		SiteID:               types.StringValue("-1"),
		Students:             types.SetNull(types.StringType),
		Teachers:             types.SetNull(types.StringType),
		StudentGroupIDs:      types.SetNull(types.StringType),
		TeacherGroupIDs:      types.SetNull(types.StringType),
		MobileDeviceGroupIDs: types.SetNull(types.StringType),
	}

	out, diags := buildClassInput(context.Background(), plan)
	if diags.HasError() {
		t.Fatalf("diagnostics: %v", diags)
	}
	if out.Students == nil || out.Students.Student == nil || len(*out.Students.Student) != 0 {
		t.Error("students wrapper must always be emitted (non-nil, empty)")
	}
	if out.Teachers == nil || out.Teachers.Teacher == nil || len(*out.Teachers.Teacher) != 0 {
		t.Error("teachers wrapper must always be emitted (non-nil, empty)")
	}
	if out.StudentGroupIds == nil || out.StudentGroupIds.ID == nil || len(*out.StudentGroupIds.ID) != 0 {
		t.Error("student_group_ids wrapper must always be emitted (non-nil, empty)")
	}
	if out.TeacherGroupIds == nil || out.TeacherGroupIds.ID == nil || len(*out.TeacherGroupIds.ID) != 0 {
		t.Error("teacher_group_ids wrapper must always be emitted (non-nil, empty)")
	}
	if out.MobileDeviceGroupIds == nil || out.MobileDeviceGroupIds.ID == nil || len(*out.MobileDeviceGroupIds.ID) != 0 {
		t.Error("mobile_device_group_ids wrapper must always be emitted (non-nil, empty)")
	}
	// student_ids / teacher_ids are resolved by the server — never sent.
	if out.StudentIds != nil || out.TeacherIds != nil {
		t.Error("student_ids / teacher_ids must not be sent (server-resolved)")
	}
	if out.Site == nil || out.Site.ID == nil || *out.Site.ID != -1 {
		t.Errorf("site sentinel -1 expected, got %v", out.Site)
	}
}

// TestBuildInput_EmptyWrappersMarshalAsEmptyElements verifies the load-bearing
// clear mechanism: an empty membership wrapper must serialise as an empty XML
// element (e.g. <students></students>), NOT be omitted. The classic PUT merges
// omitted fields, so if the empty element were dropped the server would retain
// stale members and removing them via Terraform would silently no-op.
func TestBuildInput_EmptyWrappersMarshalAsEmptyElements(t *testing.T) {
	plan := ClassResourceModel{
		Name:                 types.StringValue("clearing class"),
		SiteID:               types.StringValue("-1"),
		Students:             types.SetNull(types.StringType),
		Teachers:             types.SetNull(types.StringType),
		StudentGroupIDs:      types.SetNull(types.StringType),
		TeacherGroupIDs:      types.SetNull(types.StringType),
		MobileDeviceGroupIDs: types.SetNull(types.StringType),
	}
	out, diags := buildClassInput(context.Background(), plan)
	if diags.HasError() {
		t.Fatalf("diagnostics: %v", diags)
	}
	raw, err := xml.Marshal(out)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	xmlStr := string(raw)
	for _, elem := range []string{
		"<students></students>",
		"<teachers></teachers>",
		"<student_group_ids></student_group_ids>",
		"<teacher_group_ids></teacher_group_ids>",
		"<mobile_device_group_ids></mobile_device_group_ids>",
	} {
		if !strings.Contains(xmlStr, elem) {
			t.Errorf("expected empty element %q in marshalled payload to clear members; got:\n%s", elem, xmlStr)
		}
	}
}

func TestBuildInput_PopulatedMembership(t *testing.T) {
	plan := ClassResourceModel{
		Name:                 types.StringValue("biology"),
		SiteID:               types.StringValue("-1"),
		Students:             strSet(t, "a@x.com", "b@x.com"),
		Teachers:             strSet(t, "t@x.com"),
		StudentGroupIDs:      strSet(t, "3"),
		TeacherGroupIDs:      strSet(t, "1"),
		MobileDeviceGroupIDs: strSet(t, "66", "876"),
	}
	out, diags := buildClassInput(context.Background(), plan)
	if diags.HasError() {
		t.Fatalf("diagnostics: %v", diags)
	}
	if len(*out.Students.Student) != 2 {
		t.Errorf("expected 2 students, got %d", len(*out.Students.Student))
	}
	if len(*out.Teachers.Teacher) != 1 {
		t.Errorf("expected 1 teacher, got %d", len(*out.Teachers.Teacher))
	}
	if len(*out.MobileDeviceGroupIds.ID) != 2 {
		t.Errorf("expected 2 mobile device group ids, got %d", len(*out.MobileDeviceGroupIds.ID))
	}
}

func TestBuildInput_NonIntegerGroupID(t *testing.T) {
	plan := ClassResourceModel{
		Name:            types.StringValue("bad ids"),
		SiteID:          types.StringValue("-1"),
		StudentGroupIDs: strSet(t, "not-a-number"),
	}
	_, diags := buildClassInput(context.Background(), plan)
	if !diags.HasError() {
		t.Fatal("expected diagnostics for non-integer group id")
	}
}

func TestBuildSiteObject_NoneSentinel(t *testing.T) {
	if got := buildSiteObject(types.StringValue("-1")); got == nil || got.ID == nil || *got.ID != -1 {
		t.Errorf("expected site id -1, got %v", got)
	}
	if got := buildSiteObject(types.StringNull()); got != nil {
		t.Errorf("null site_id should produce nil SiteObject, got %v", got)
	}
}
