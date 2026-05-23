// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package printer

import (
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// buildPrinterInput converts the Terraform plan model into the SDK Printer
// payload used for both Create and Update. ID is omitted on write — Create
// uses path id="0" and Update derives it from state.
//
// Two fields use non-default emission semantics confirmed by the §13.2 audit:
//
//   - `category` is emitted on every write (stringPtrEmitAlways). The
//     classic endpoint treats an omitted `<category>` tag as "preserve
//     current" and an empty `<category></category>` as "clear to the
//     sentinel." TF state semantics require a null `category` to mean
//     "unassigned", so we always emit and let the server map empty to
//     `categoryUnassignedSentinel`.
//
//   - `ppd_path`, `ppd`, and `ppd_contents` are emitted only when set. When
//     `use_generic = true` the server clears `ppd`/`ppd_contents` and forces
//     `ppd_path` to the system Generic.ppd path regardless of input — the
//     ConfigValidator blocks the user from setting these in that combination,
//     so a null model value means "do not touch" and must omit the tag.
func buildPrinterInput(plan PrinterResourceModel) *proclassic.Printer {
	return &proclassic.Printer{
		Name:           helpers.OptionalStringPointer(plan.Name),
		Category:       stringPtrEmitAlways(plan.Category),
		URI:            helpers.OptionalStringPointer(plan.URI),
		CUPSName:       helpers.OptionalStringPointer(plan.CUPSName),
		Location:       helpers.OptionalStringPointer(plan.Location),
		Model:          helpers.OptionalStringPointer(plan.Model),
		Info:           helpers.OptionalStringPointer(plan.Info),
		Notes:          helpers.OptionalStringPointer(plan.Notes),
		MakeDefault:    optionalBoolPointer(plan.MakeDefault),
		UseGeneric:     optionalBoolPointer(plan.UseGeneric),
		Ppd:            helpers.OptionalStringPointer(plan.PPD),
		PpdPath:        helpers.OptionalStringPointer(plan.PPDPath),
		PpdContents:    helpers.OptionalStringPointer(plan.PPDContents.StringValue),
		Shared:         stringPtrFromBool(plan.Shared),
		OsRequirements: helpers.OptionalStringPointer(plan.OSRequirements),
	}
}

// optionalBoolPointer returns a *bool for a TF Bool, or nil when null/unknown.
// Mirrors helpers.OptionalStringPointer's contract for booleans — the SDK
// omitempty tag drops the field from the wire when the pointer is nil.
func optionalBoolPointer(v interface {
	IsNull() bool
	IsUnknown() bool
	ValueBool() bool
}) *bool {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	out := v.ValueBool()
	return &out
}
