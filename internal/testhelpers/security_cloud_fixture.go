// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package testhelpers

import (
	"context"
	"sync"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/securitycloud"
)

// sharedGatewaysOnce caches the shared-gateway read so the acceptance suite hits
// the endpoint at most once per run.
var sharedGatewaysOnce sync.Once

// sharedGateways holds the cached shared-gateway list.
var sharedGateways []securitycloud.SharedGateway

// sharedGatewaysErr captures the read failure, if any.
var sharedGatewaysErr error

// RequireSecurityCloudSharedGatewayIDs returns at least count Jamf-managed shared
// ZTNA gateway IDs for tests that need real gateway references.
//
// Shared gateways are a Jamf-managed catalog — every entitled tenant has the same
// set ("Nearest Data Center" plus a shared IP pool per region), so this needs no
// fixture creation and nothing to clean up. Custom DNS zone name servers reference
// them by ID, and the reference is server-enforced: a zone naming a gateway that
// does not exist is refused, so a test cannot invent an ID.
//
// A read failure or a tenant with fewer than count gateways SKIPS rather than
// fails. A tenant without the Security Cloud entitlement is a legitimate
// environment for the rest of the suite, and inferring "the resource is broken"
// from absent tenant data is exactly the wrong conclusion — the resource's own
// behaviour is guarded by unit tests instead.
func RequireSecurityCloudSharedGatewayIDs(t *testing.T, count int) []string {
	t.Helper()

	sharedGatewaysOnce.Do(func() {
		c, err := initAcceptanceClient()
		if err != nil {
			sharedGatewaysErr = err
			return
		}
		list, listErr := securitycloud.New(c).ListZtnaSharedGatewaysV1(context.Background())
		if listErr != nil {
			sharedGatewaysErr = listErr
			return
		}
		sharedGateways = list.Results
	})

	if sharedGatewaysErr != nil {
		t.Skipf("Skipping: cannot read Jamf Security Cloud shared ZTNA gateways (tenant may lack the entitlement): %v", sharedGatewaysErr)
	}
	if len(sharedGateways) < count {
		t.Skipf("Skipping: tenant exposes %d shared ZTNA gateway(s), test needs %d", len(sharedGateways), count)
	}

	ids := make([]string, 0, count)
	for _, g := range sharedGateways[:count] {
		ids = append(ids, g.ID)
	}
	return ids
}
