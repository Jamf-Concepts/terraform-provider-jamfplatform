// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package scope

import (
	"context"
	"strconv"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// FlattenIDNameSet projects a ProClassic []IDName pointer-slice into a Terraform
// Set<String> of the items' integer IDs. Thin, diagnostics-dropping wrapper over
// FlattenIDSlice for the common proclassic.IDName shape that classic scope and
// reference sub-blocks use throughout. Returns a null set for nil/empty input.
func FlattenIDNameSet(ctx context.Context, items *[]proclassic.IDName) types.Set {
	out, _ := FlattenIDSlice(ctx, items, func(i proclassic.IDName) *int { return i.ID })
	return out
}

// FlattenNameSet projects a ProClassic []IDName pointer-slice into a Terraform
// Set<String> of the items' names. Name-only sibling of FlattenIDNameSet.
func FlattenNameSet(ctx context.Context, items *[]proclassic.IDName) types.Set {
	out, _ := FlattenNameSlice(ctx, items, func(i proclassic.IDName) *string { return i.Name })
	return out
}

// FlattenSiteObject unpacks a ProClassic SiteObject (nil-able integer ID + name)
// into (idString, name) pointers for state assignment. Returns (nil, nil) when the
// site is absent; the ID is rendered as its decimal string when present.
func FlattenSiteObject(site *proclassic.SiteObject) (*string, *string) {
	if site == nil {
		return nil, nil
	}
	var idPtr *string
	if site.ID != nil {
		s := strconv.Itoa(*site.ID)
		idPtr = &s
	}
	return idPtr, site.Name
}

// BuildSiteObject parses a Terraform string site ID into a ProClassic SiteObject
// for write payloads. Returns nil for null/unknown/empty/un-parseable input so the
// SDK omits the site element entirely.
func BuildSiteObject(siteID types.String) *proclassic.SiteObject {
	if siteID.IsNull() || siteID.IsUnknown() {
		return nil
	}
	idStr := siteID.ValueString()
	if idStr == "" {
		return nil
	}
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return nil
	}
	return &proclassic.SiteObject{ID: &id}
}
