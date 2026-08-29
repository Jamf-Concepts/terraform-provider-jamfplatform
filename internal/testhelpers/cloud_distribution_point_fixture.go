// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package testhelpers

import (
	"context"
	"strings"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
)

// EnsurePrincipalCloudDistributionPoint makes the tenant's existing cloud
// distribution point the principal (master) DP and restores the original
// `master` flag on t.Cleanup. Policies that reference the "default" distribution
// point in their package configuration cannot be saved unless a principal DP is
// configured (the server otherwise rejects the write with 409 "Problem with
// distribution server").
//
// It NEVER creates or deletes the cloud distribution point: deleting one
// permanently wipes every package, in-house app, and eBook hosted in Jamf Cloud
// (see the cloud_distribution_point resource docs). It only PATCHes the `master`
// flag, which is non-destructive and reversible.
//
// The calling test is skipped when no cloud distribution point is configured
// (cdn_type NONE) — there is nothing to make principal, and standing one up
// would be destructive to tear down.
func EnsurePrincipalCloudDistributionPoint(t *testing.T) {
	t.Helper()
	client := pro.New(NewAcceptanceClient(t))
	ctx := context.Background()

	got, err := client.GetCloudDistributionPointV1(ctx)
	if err != nil {
		t.Fatalf("reading cloud distribution point: %v", err)
	}
	if got == nil || strings.EqualFold(got.CdnType, pro.CloudDistributionPointCdnTypeNone) {
		t.Skip("skipping: no cloud distribution point configured on tenant; cannot make one principal without a destructive create")
	}

	if got.Master != nil && *got.Master {
		return // already principal — nothing to change, nothing to restore
	}

	if _, err := client.UpdateCloudDistributionPointV1(ctx, cdpMasterPatch(got, true)); err != nil {
		t.Fatalf("setting cloud distribution point as principal: %v", err)
	}
	t.Cleanup(func() {
		_, _ = client.UpdateCloudDistributionPointV1(context.Background(), cdpMasterPatch(got, false))
	})
}

// cdpMasterPatch builds the minimal merge-patch that flips only the master flag.
// cdn_type and username are echoed from the current record (cdn_type is
// mandatory in every PATCH body); password is write-only, never returned, and
// not needed for JAMF_CLOUD, so it is left empty.
func cdpMasterPatch(current *pro.CloudDistributionPoint, master bool) *pro.CloudDistributionPoint {
	return &pro.CloudDistributionPoint{
		CdnType:  current.CdnType,
		Username: current.Username,
		Master:   &master,
	}
}
