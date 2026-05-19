// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// Package providerdata defines the value passed from the provider Configure phase to
// every resource, data source, list resource, and action Configure call. It is in its
// own package (rather than internal/provider) to avoid an import cycle: internal/provider
// imports every resource package for registration, so resource packages cannot import it.
package providerdata

import (
	"context"
	"sync"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform"
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-framework/diag"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// ProviderMinJamfProVersion is the provider-wide recommended minimum Jamf Pro tenant
// version. Surfaces as a warning (not an error) when the tenant version is below it.
// Bump at release time by grepping all minJamfProVersion constants under
// internal/resources/pro/ and taking the max.
const ProviderMinJamfProVersion = "11.27.0"

// Data is the value passed via ResourceData/DataSourceData/ListResourceData/ActionData.
// It bundles the authenticated SDK client with lazy Jamf Pro version state shared
// across all Pro resource Configure calls in a single terraform invocation.
type Data struct {
	Client *jamfplatform.Client

	proVersionOnce sync.Once
	proVersion     string
	proVersionErr  error

	warnOnce             sync.Once
	providerFloorWarning diag.Diagnostic
}

// New wraps a configured SDK client in a Data value.
func New(client *jamfplatform.Client) *Data {
	return &Data{Client: client}
}

// GetJamfProVersion fetches the tenant's Jamf Pro version using GetJamfProVersionV1,
// caching the result for the lifetime of the Data value. Subsequent calls return the
// cached value (or cached error). Resources with empty minJamfProVersion should not
// call this — fetching only fires when a Pro resource with a version requirement is
// in the config.
func (d *Data) GetJamfProVersion(ctx context.Context) (string, error) {
	d.proVersionOnce.Do(func() {
		v, err := pro.New(d.Client).GetJamfProVersionV1(ctx)
		if err != nil {
			d.proVersionErr = err
			return
		}
		if v != nil {
			d.proVersion = v.Version
		}
	})
	return d.proVersion, d.proVersionErr
}

// MaybeProviderFloorWarning returns a warning diagnostic if the cached tenant version is
// below ProviderMinJamfProVersion. Computed exactly once per Data lifetime; subsequent
// calls return the same diagnostic. Returns nil when at/above floor or when no Pro
// version has been fetched yet.
func (d *Data) MaybeProviderFloorWarning() diag.Diagnostic {
	d.warnOnce.Do(func() {
		if d.proVersion == "" {
			return
		}
		d.providerFloorWarning = helpers.WarnIfBelowProviderFloor(d.proVersion, ProviderMinJamfProVersion)
	})
	return d.providerFloorWarning
}
