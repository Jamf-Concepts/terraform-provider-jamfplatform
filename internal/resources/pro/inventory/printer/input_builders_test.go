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
	if got.Shared == nil || *got.Shared != "true" {
		t.Errorf("expected Shared=\"true\", got %v", got.Shared)
	}
	if got.OsRequirements == nil || *got.OsRequirements != "13.5, 16.0" {
		t.Errorf("expected OsRequirements, got %v", got.OsRequirements)
	}
	if got.ID != nil {
		t.Errorf("expected nil ID on write payload, got %d", *got.ID)
	}
}

// TestBuildPrinterInput_CategoryAlwaysEmittedWhenNull verifies the special
// "always emit" encoding for category. Null in TF state must serialize to a
// non-nil *string (pointing at "") so the SDK emits `<category></category>`
// and the server clears the category to its sentinel. Returning nil instead
// would omit the tag, which the classic endpoint treats as "preserve current"
// — the exact opposite semantic.
func TestBuildPrinterInput_CategoryAlwaysEmittedWhenNull(t *testing.T) {
	plan := PrinterResourceModel{
		Name:     types.StringValue("Front Desk"),
		Category: types.StringNull(),
	}
	got := buildPrinterInput(plan)

	if got.Category == nil {
		t.Fatalf("Category must serialize to a non-nil *string even when null (sends empty <category/>); got nil")
	}
	if *got.Category != "" {
		t.Errorf("null Category must emit empty string, got %q", *got.Category)
	}
}

// TestBuildPrinterInput_PPDPathOmittedWhenNull verifies that null ppd_path
// produces a nil pointer (omitted tag) — the opposite of category. Emitting
// an empty tag here would clear the server's auto-fill under use_generic=true.
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
	if got.Ppd != nil {
		t.Errorf("null Ppd must serialise to nil, got %q", *got.Ppd)
	}
}

// TestBuildPrinterInput_SharedFalseEmits verifies that explicit shared=false
// (as opposed to null) reaches the wire as "false" rather than nil — the user
// can deliberately disable shared even when the schema default is false.
func TestBuildPrinterInput_SharedFalseEmits(t *testing.T) {
	plan := PrinterResourceModel{
		Name:   types.StringValue("Lab"),
		Shared: types.BoolValue(false),
	}
	got := buildPrinterInput(plan)

	if got.Shared == nil || *got.Shared != "false" {
		t.Errorf("expected Shared=\"false\", got %v", got.Shared)
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

func TestBuildPrinterInput_MostFieldsNullOmitted(t *testing.T) {
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

	// Name is the only required field set. Category gets the always-emit
	// treatment. Everything else must omit.
	if got.Name == nil || *got.Name != "Minimal" {
		t.Errorf("expected Name=Minimal, got %v", got.Name)
	}
	if got.Category == nil || *got.Category != "" {
		t.Errorf("expected Category emitted as empty, got %v", got.Category)
	}
	for _, c := range []struct {
		name string
		ptr  any
	}{
		{"URI", got.URI},
		{"CUPSName", got.CUPSName},
		{"Location", got.Location},
		{"Model", got.Model},
		{"Info", got.Info},
		{"Notes", got.Notes},
		{"MakeDefault", got.MakeDefault},
		{"UseGeneric", got.UseGeneric},
		{"Ppd", got.Ppd},
		{"PpdPath", got.PpdPath},
		{"PpdContents", got.PpdContents},
		{"Shared", got.Shared},
		{"OsRequirements", got.OsRequirements},
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
