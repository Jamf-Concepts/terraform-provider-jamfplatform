// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package testhelpers

import (
	"context"
	"os"
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

// RequireSecurityCloudTenantID returns the tenant ID a Security Cloud gateway must
// be granted access to.
//
// `tenantIds` is required on every gateway and grouped gateway, and Jamf Security
// Cloud validates each entry against the caller's organization: an ID outside it
// is refused with `400 BAD_REQUEST` ("No mapping found for one of the supplied
// ids"). So the value cannot be invented — it has to be the tenant the provider is
// scoped to, which AccPreCheckSecurityCloud has already established the operator
// declared.
//
// Under an environment-scoped integration there is no single tenant ID to use, and
// nothing readable to derive one from, so the test skips. That is the honest
// outcome rather than guessing: a gateway test that cannot name a tenant has
// nothing to assert.
func RequireSecurityCloudTenantID(t *testing.T) string {
	t.Helper()
	tenantID := os.Getenv("JAMFPLATFORM_SECURITY_CLOUD_TENANT_ID")
	if tenantID == "" {
		t.Skip("Skipping: JAMFPLATFORM_SECURITY_CLOUD_TENANT_ID must name the tenant a ZTNA gateway grants access to; an environment-scoped run has no single tenant ID to use")
	}
	return tenantID
}

// contentCategoriesOnce caches the content-category read so the acceptance suite
// hits the endpoint at most once per run.
var contentCategoriesOnce sync.Once

// contentCategories holds the cached content-category list.
var contentCategories []securitycloud.Category

// contentCategoriesErr captures the read failure, if any.
var contentCategoriesErr error

// RequireSecurityCloudContentCategories returns the Jamf-curated content category
// catalogue for tests that need a real category reference.
//
// The catalogue is Jamf's, not the tenant's — identical for every entitled tenant
// and not writable, so there is nothing to create and nothing to clean up. A read
// failure or an empty catalogue SKIPS rather than fails, for the same reason
// RequireSecurityCloudSharedGatewayIDs does: a tenant without the entitlement is a
// legitimate environment, and concluding "the data source is broken" from absent
// tenant data is the wrong inference.
func RequireSecurityCloudContentCategories(t *testing.T) []securitycloud.Category {
	t.Helper()

	contentCategoriesOnce.Do(func() {
		c, err := initAcceptanceClient()
		if err != nil {
			contentCategoriesErr = err
			return
		}
		list, listErr := securitycloud.New(c).ListContentCategoriesV1(context.Background())
		if listErr != nil {
			contentCategoriesErr = listErr
			return
		}
		contentCategories = list.Results
	})

	if contentCategoriesErr != nil {
		t.Skipf("Skipping: cannot read Jamf Security Cloud content categories (tenant may lack the entitlement): %v", contentCategoriesErr)
	}
	if len(contentCategories) == 0 {
		t.Skip("Skipping: tenant exposes no Jamf Security Cloud content categories")
	}
	return contentCategories
}

// predefinedAppsOnce caches the predefined-app read so the acceptance suite hits
// the endpoint at most once per run.
var predefinedAppsOnce sync.Once

// predefinedApps holds the cached predefined-app list.
var predefinedApps []securitycloud.PredefinedApp

// predefinedAppsErr captures the read failure, if any.
var predefinedAppsErr error

// RequireSecurityCloudPredefinedApps returns the Jamf-curated Zero Trust Network
// Access app templates for tests that need a real template reference.
//
// Same contract as RequireSecurityCloudContentCategories: a Jamf-managed catalogue,
// nothing to create, and absence skips rather than fails.
func RequireSecurityCloudPredefinedApps(t *testing.T) []securitycloud.PredefinedApp {
	t.Helper()

	predefinedAppsOnce.Do(func() {
		c, err := initAcceptanceClient()
		if err != nil {
			predefinedAppsErr = err
			return
		}
		list, listErr := securitycloud.New(c).ListZtnaPredefinedAppsV1(context.Background())
		if listErr != nil {
			predefinedAppsErr = listErr
			return
		}
		predefinedApps = list.Results
	})

	if predefinedAppsErr != nil {
		t.Skipf("Skipping: cannot read Jamf Security Cloud predefined ZTNA apps (tenant may lack the entitlement): %v", predefinedAppsErr)
	}
	if len(predefinedApps) == 0 {
		t.Skip("Skipping: tenant exposes no Jamf Security Cloud predefined ZTNA apps")
	}
	return predefinedApps
}
