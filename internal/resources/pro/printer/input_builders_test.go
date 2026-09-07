// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package printer

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBuildPrinterInput_AllFieldsSet(t *testing.T) {
	plan := PrinterResourceModel{
		Name:           types.StringValue("Lab Color"),
		Category:       types.StringValue("Printers"),
		URI:            types.StringValue("ipp://printer.lab.example.com/queue1"),
		CUPSName:       types.StringValue("lab_color"),
		Location:       types.StringValue("Building 5, floor 2"),
		Model:          types.StringValue("HP DeskJet 2600 series"),
		Info:           types.StringValue("Drop sheets at the loading bay."),
		Notes:          types.StringValue("Created 2026."),
		MakeDefault:    types.BoolValue(true),
		UseGeneric:     types.BoolValue(false),
		PPD:            types.StringValue("HP DeskJet 2600 series.ppd"),
		PPDPath:        types.StringValue("/Library/Printers/PPDs/Contents/Resources/HP DeskJet 2600 series.ppd"),
		PPDContents:    newTrimmedStringValue("*PPD-Adobe: \"4.3\""),
		Shared:         types.BoolValue(true),
		OSRequirements: types.StringValue("13.5, 16.0"),
	}
	got := buildPrinterInput(plan)

	if got.Name == nil || *got.Name != "Lab Color" {
		t.Errorf("expected Name=Lab Color, got %v", got.Name)
	}
	if got.Category == nil || *got.Category != "Printers" {
		t.Errorf("expected Category=Printers, got %v", got.Category)
	}
	if got.URI == nil || *got.URI != "ipp://printer.lab.example.com/queue1" {
		t.Errorf("expected URI, got %v", got.URI)
	}
	if got.CUPSName == nil || *got.CUPSName != "lab_color" {
		t.Errorf("expected CUPSName, got %v", got.CUPSName)
	}
	if got.MakeDefault == nil || !*got.MakeDefault {
		t.Errorf("expected MakeDefault=true, got %v", got.MakeDefault)
	}
	if got.UseGeneric == nil || *got.UseGeneric {
		t.Errorf("expected UseGeneric=false, got %v", got.UseGeneric)
	}
	if got.PpdPath == nil || *got.PpdPath != "/Library/Printers/PPDs/Contents/Resources/HP DeskJet 2600 series.ppd" {
		t.Errorf("expected PpdPath, got %v", got.PpdPath)
	}
	if got.Shared == nil || !*got.Shared {
		t.Errorf("expected Shared=true, got %v", got.Shared)
	}
	if got.OsRequirements == nil || *got.OsRequirements != "13.5, 16.0" {
		t.Errorf("expected OsRequirements, got %v", got.OsRequirements)
	}
	if got.ID != nil {
		t.Errorf("expected nil ID on write payload, got %d", *got.ID)
	}
}

// TestBuildPrinterInput_ClearableStringsAlwaysEmittedWhenNull verifies the
// always-emit encoding for every plain Optional string. Null in the plan must
// serialize to a non-nil *string pointing at "" so the SDK emits an empty
// element and the classic merge clears the stored value; returning nil would
// omit the tag, which the endpoint treats as "preserve current" — the exact
// opposite semantic, and the cause of issue #384.
func TestBuildPrinterInput_ClearableStringsAlwaysEmittedWhenNull(t *testing.T) {
	plan := PrinterResourceModel{
		Name:           types.StringValue("Front Desk"),
		Category:       types.StringNull(),
		URI:            types.StringNull(),
		CUPSName:       types.StringNull(),
		Location:       types.StringNull(),
		Model:          types.StringNull(),
		Info:           types.StringNull(),
		Notes:          types.StringNull(),
		PPD:            types.StringNull(),
		OSRequirements: types.StringNull(),
	}
	got := buildPrinterInput(plan)

	for _, c := range []struct {
		name string
		ptr  *string
	}{
		{"Category", got.Category},
		{"URI", got.URI},
		{"CUPSName", got.CUPSName},
		{"Location", got.Location},
		{"Model", got.Model},
		{"Info", got.Info},
		{"Notes", got.Notes},
		{"Ppd", got.Ppd},
		{"OsRequirements", got.OsRequirements},
	} {
		if c.ptr == nil {
			t.Errorf("%s must serialize to a non-nil *string even when null (sends an empty element); got nil", c.name)
			continue
		}
		if *c.ptr != "" {
			t.Errorf("null %s must emit empty string, got %q", c.name, *c.ptr)
		}
	}
}

// TestBuildPrinterInput_PPDPathOmittedWhenNull verifies that null ppd_path
// produces a nil pointer (omitted tag) — the opposite of ppd. Emitting an
// empty tag here would clear the server's auto-fill under use_generic=true.
func TestBuildPrinterInput_PPDPathOmittedWhenNull(t *testing.T) {
	plan := PrinterResourceModel{
		Name:    types.StringValue("Front Desk"),
		PPDPath: types.StringNull(),
		PPD:     types.StringNull(),
	}
	got := buildPrinterInput(plan)

	if got.PpdPath != nil {
		t.Errorf("null PpdPath must serialise to nil (omitted tag preserves server auto-fill), got %q", *got.PpdPath)
	}
	if got.PpdContents != nil {
		t.Errorf("null PpdContents must serialise to nil, got %q", *got.PpdContents)
	}
}

// TestBuildPrinterInput_SharedFalseEmits verifies that explicit shared=false
// (as opposed to null) reaches the wire as false rather than nil — the user
// can deliberately disable shared even when the schema default is false.
func TestBuildPrinterInput_SharedFalseEmits(t *testing.T) {
	plan := PrinterResourceModel{
		Name:   types.StringValue("Lab"),
		Shared: types.BoolValue(false),
	}
	got := buildPrinterInput(plan)

	if got.Shared == nil || *got.Shared {
		t.Errorf("expected Shared=false, got %v", got.Shared)
	}
}

func TestBuildPrinterInput_SharedNullOmitted(t *testing.T) {
	plan := PrinterResourceModel{
		Name:   types.StringValue("Lab"),
		Shared: types.BoolNull(),
	}
	got := buildPrinterInput(plan)

	if got.Shared != nil {
		t.Errorf("null Shared must serialise to nil, got %v", *got.Shared)
	}
}

// TestBuildPrinterInput_ServerOwnedFieldsNullOmitted pins the fields that
// must still omit when null: the Optional+Computed bools and the two PPD
// fields whose null means "server-owned" (see buildPrinterInput).
func TestBuildPrinterInput_ServerOwnedFieldsNullOmitted(t *testing.T) {
	plan := PrinterResourceModel{
		Name:           types.StringValue("Minimal"),
		Category:       types.StringNull(),
		URI:            types.StringNull(),
		CUPSName:       types.StringNull(),
		Location:       types.StringNull(),
		Model:          types.StringNull(),
		Info:           types.StringNull(),
		Notes:          types.StringNull(),
		MakeDefault:    types.BoolNull(),
		UseGeneric:     types.BoolNull(),
		PPD:            types.StringNull(),
		PPDPath:        types.StringNull(),
		PPDContents:    newTrimmedStringNull(),
		Shared:         types.BoolNull(),
		OSRequirements: types.StringNull(),
	}
	got := buildPrinterInput(plan)

	if got.Name == nil || *got.Name != "Minimal" {
		t.Errorf("expected Name=Minimal, got %v", got.Name)
	}
	for _, c := range []struct {
		name string
		ptr  any
	}{
		{"MakeDefault", got.MakeDefault},
		{"UseGeneric", got.UseGeneric},
		{"PpdPath", got.PpdPath},
		{"PpdContents", got.PpdContents},
		{"Shared", got.Shared},
	} {
		if !isNilPointer(c.ptr) {
			t.Errorf("%s must be nil when TF value null, got non-nil", c.name)
		}
	}
}

// isNilPointer reports whether an `any` containing a typed pointer is nil.
// The fields under test are *string and *bool; an `interface{}(*T)(nil)` is
// not equal to literal nil with ==, so we extract the value and check it.
func isNilPointer(v any) bool {
	switch p := v.(type) {
	case *string:
		return p == nil
	case *bool:
		return p == nil
	default:
		return v == nil
	}
}
