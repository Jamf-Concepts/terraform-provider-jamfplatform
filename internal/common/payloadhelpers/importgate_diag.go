// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package payloadhelpers

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// ImportGateSummary is the diagnostic summary both configuration-profile
// resources use when the import gate refuses a profile.
const ImportGateSummary = "This profile cannot be managed by Terraform"

// ImportGateDiagnostics returns an error diagnostic when Jamf Pro would mangle
// this payload on write-back, and no diagnostics otherwise. Callers must invoke
// it ONLY on the import path — never on ordinary refresh, where blocking would
// strand an already-managed or externally-corrupted resource with no way to see
// drift or remove it.
//
// An empty or unparseable payload yields no diagnostics: there is nothing to
// predict, and the post-write checks on create/update still backstop it.
// label identifies the profile in the detail text. Terraform does not print the
// resource address alongside a Read error raised during import config
// generation, so without this the operator is told a profile is unmanageable but
// not which one — useless when importing a whole tenant at once.
func ImportGateDiagnostics(payload []byte, platform ProfilePlatform, name, id string) diag.Diagnostics {
	var diags diag.Diagnostics
	if len(payload) == 0 {
		return diags
	}
	detail, lossy, ok := PayloadImportRisk(payload, platform)
	if !ok || !lossy {
		return diags
	}
	diags.AddError(ImportGateSummary, profileLabel(name, id)+detail)
	return diags
}

// profileLabel renders the leading identification line, omitting whichever of
// name/id is unavailable.
func profileLabel(name, id string) string {
	switch {
	case name != "" && id != "":
		return fmt.Sprintf("Profile: %s (id %s)\n\n", name, id)
	case name != "":
		return fmt.Sprintf("Profile: %s\n\n", name)
	case id != "":
		return fmt.Sprintf("Profile id: %s\n\n", id)
	default:
		return ""
	}
}

// ImportGateSkip reports whether a list item must be left out of generated
// config. It is the list-resource counterpart to ImportGateDiagnostics: config
// generation runs over every profile in the tenant, so one unmanageable profile
// must not abort the whole stream.
//
// A skipped item is dropped from the stream entirely rather than streamed with a
// nil Resource. The framework documents a nil Resource under IncludeResource as
// merely warning, but Terraform CLI 1.15 panics in its genconfig walk when it
// tries to render one (cty GetAttr on a null object), so the identity-only shape
// is not usable here.
func ImportGateSkip(payload []byte, platform ProfilePlatform) bool {
	if len(payload) == 0 {
		return false
	}
	_, lossy, ok := PayloadImportRisk(payload, platform)
	return ok && lossy
}

// ImportGateSkipWarning renders the single consolidated warning for every item
// ImportGateSkip dropped, to be attached to a streamed result (or to the stream
// itself when nothing was emitted). One diagnostic naming all of them beats N
// full diagnostics: the per-profile detail is available by importing that profile
// on its own, which is the next thing the operator does anyway.
//
// Returns no diagnostics when nothing was skipped.
func ImportGateSkipWarning(skipped []string, resourceType string) diag.Diagnostics {
	var diags diag.Diagnostics
	if len(skipped) == 0 {
		return diags
	}

	noun := "profile"
	if len(skipped) > 1 {
		noun = "profiles"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "No configuration was generated for the following %s, because Jamf Pro cannot store %s payload back unchanged and the first apply that touched %s would corrupt it irreversibly:\n\n",
		noun, map[bool]string{true: "their", false: "its"}[len(skipped) > 1], map[bool]string{true: "them", false: "it"}[len(skipped) > 1])
	shown := skipped
	if len(shown) > maxListedSkips {
		shown = shown[:maxListedSkips]
	}
	for _, name := range shown {
		fmt.Fprintf(&b, "  - %s\n", name)
	}
	if len(skipped) > len(shown) {
		fmt.Fprintf(&b, "  … and %d more\n", len(skipped)-len(shown))
	}
	fmt.Fprintf(&b, "\nEvery other profile was generated as normal. To see exactly which payload value is at fault for one of these, import it on its own — `terraform import %s.example <id>` — and the error names the value and the fix.",
		resourceType)

	diags.AddWarning(
		fmt.Sprintf("Skipped %d %s that Terraform cannot manage", len(skipped), noun),
		b.String(),
	)
	return diags
}

// maxListedSkips bounds the consolidated warning: a tenant-wide query on a large
// estate could otherwise name hundreds of profiles in one diagnostic.
const maxListedSkips = 20
