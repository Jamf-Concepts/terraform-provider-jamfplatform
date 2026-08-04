// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// Package computertarget resolves a Jamf Pro computer inventory id from the
// serial-number / management-id / udid selector shared by device-targeting
// actions (e.g. redeploy_management_framework, jamf_protect_deployment_retry).
// The device data sources surface a management id (UUID), not the Jamf Pro id,
// so a management id is mapped through the computer inventory; a serial number
// and a hardware UDID each resolve directly.
//
// # Endpoint status
//
// Every version of /computers-inventory — V1, V2 and V3 — is deprecated in the
// SDK's bundled 11.30.0 spec as of 2026-07-14. A live Jamf Pro 11.30.2 tenant
// serves /v4/computers-inventory with no Deprecation header and an identical
// RSQL filter contract (wire-probed 2026-08-04: filtering on general.managementId
// returns the same single record), but /v4 is absent from the bundled spec, so
// the SDK generates no client for it. Tracked upstream as
// Jamf-Concepts/jamfplatform-go-sdk#50.
//
// Migration window per STYLE_GUIDE §Deprecation migration timeline: 6-month soft
// target 2027-01-14, hard floor 3 months before Jamf's announced removal date.
// Migrating means swapping ListComputersInventoryV3 for its V4 equivalent and
// dropping the SA1019 suppression below — a one-line change once the SDK exposes
// it, which is why this waits for the SDK rather than routing around it.
//
// Deliberately NOT migrated to /preview/computers, which is undeprecated and
// does carry managementId: it accepts no filter, so resolving one computer would
// mean paging the entire computer list. That trades a tracked deprecation for an
// unbounded per-invocation cost that grows with fleet size.
package computertarget

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// ResolveComputerID ensures exactly one computer identifier is provided and
// returns the Jamf Pro computer id (an integer string). It reports its progress
// and any failure through resp; the boolean is false when the caller should
// stop (a diagnostic has been added).
func ResolveComputerID(ctx context.Context, client *pro.Client, resp *action.InvokeResponse, managementIDAttr, serialNumberAttr, udidAttr types.String) (string, bool) {
	hasID := helpers.IsConfiguredValue(managementIDAttr)
	hasSerial := helpers.IsConfiguredValue(serialNumberAttr)
	hasUDID := helpers.IsConfiguredValue(udidAttr)

	n := 0
	for _, on := range []bool{hasID, hasSerial, hasUDID} {
		if on {
			n++
		}
	}

	switch {
	case n > 1:
		resp.Diagnostics.AddError(
			"Multiple Device Identifiers Provided",
			"Specify only one of management_id, serial_number, or udid when invoking this action.",
		)
		return "", false
	case hasID:
		managementID := managementIDAttr.ValueString()
		resp.SendProgress(action.InvokeProgressEvent{Message: fmt.Sprintf("Resolving management id %s", managementID)})

		// SA1019 suppressed: /v3/computers-inventory is spec-deprecated as of
		// 2026-07-14 and there is no generated successor yet. The two sibling
		// branches below reach the same deprecated surface through SDK resolve
		// helpers, which carry no Deprecated marker of their own — so migrating
		// this one call in isolation would buy nothing. See the package doc.
		matches, err := client.ListComputersInventoryV3(ctx, []string{"GENERAL"}, nil, fmt.Sprintf("general.managementId==%q", managementID)) //nolint:staticcheck // SA1019: no v4 client generated yet — see the package doc
		if err != nil {
			resp.Diagnostics.AddError(
				"Computer Lookup Failed",
				fmt.Sprintf("Unable to look up computer for management id %s: %s", managementID, err),
			)
			return "", false
		}
		if len(matches) == 0 || matches[0].ID == "" {
			resp.Diagnostics.AddError(
				"Computer Not Found",
				fmt.Sprintf("No computer matched management id %s.", managementID),
			)
			return "", false
		}
		return matches[0].ID, true
	case hasSerial:
		serial := serialNumberAttr.ValueString()
		resp.SendProgress(action.InvokeProgressEvent{Message: fmt.Sprintf("Resolving serial number %s", serial)})

		id, err := client.ResolveComputerInventoryV3IDBySerialNumber(ctx, serial)
		if err != nil {
			resp.Diagnostics.AddError(
				"Computer Lookup Failed",
				fmt.Sprintf("Unable to resolve serial number %s to a computer id: %s", serial, err),
			)
			return "", false
		}
		return id, true
	case hasUDID:
		udid := udidAttr.ValueString()
		resp.SendProgress(action.InvokeProgressEvent{Message: fmt.Sprintf("Resolving udid %s", udid)})

		id, err := client.ResolveComputerInventoryV3IDByUDID(ctx, udid)
		if err != nil {
			resp.Diagnostics.AddError(
				"Computer Lookup Failed",
				fmt.Sprintf("Unable to resolve udid %s to a computer id: %s", udid, err),
			)
			return "", false
		}
		return id, true
	default:
		resp.Diagnostics.AddError(
			"Missing Device Identifier",
			"Specify one of management_id, serial_number, or udid to select the computer.",
		)
		return "", false
	}
}
