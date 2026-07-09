// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mcx_forced_payload

import (
	"fmt"
	"strings"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/functions/mobileconfig"
)

// renderMCXForcedPayload builds a complete .mobileconfig plist that delivers the
// given preference domain's settings as forced (MCX) managed preferences — the
// com.apple.ManagedClient.preferences "Custom Settings" envelope.
//
// It is a thin convenience wrapper over mobileconfig.Assemble: it constructs the
// single ManagedClient.preferences payload and delegates identity injection,
// number normalization, and plist encoding to the shared assembler. The
// preference keys are free-form per-app settings, passed through verbatim.
func renderMCXForcedPayload(domain string, prefs map[string]any) ([]byte, error) {
	if strings.TrimSpace(domain) == "" {
		return nil, fmt.Errorf("preference_domain must not be empty")
	}
	if len(prefs) == 0 {
		return nil, fmt.Errorf("preferences must contain at least one key")
	}

	payload := map[string]any{
		"PayloadType":        "com.apple.ManagedClient.preferences",
		"PayloadDisplayName": "Custom Settings",
		"PayloadContent": map[string]any{
			domain: map[string]any{
				"Forced": []any{
					map[string]any{"mcx_preference_settings": prefs},
				},
			},
		},
	}

	return mobileconfig.Assemble(mobileconfig.Profile{
		DisplayName: domain,
		Identifier:  domain,
		Payloads:    []map[string]any{payload},
	})
}
