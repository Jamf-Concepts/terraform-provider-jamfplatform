// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// Package providerdata defines the value passed from the provider Configure phase to
// every resource, data source, list resource, and action Configure call. It is in its
// own package (rather than internal/provider) to avoid an import cycle: internal/provider
// imports every resource package for registration, so resource packages cannot import it.
package providerdata

import (
	"context"
	"fmt"
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
//
// Caching semantics:
//   - Successful version fetches are cached for the lifetime of the Data value.
//   - Errors are NOT cached — subsequent Configure calls will retry the fetch. This
//     avoids a transient network/auth blip in the first Pro Configure poisoning every
//     later Configure on the same terraform invocation.
//   - The provider-floor advisory warning is emitted at most once per Data value to
//     prevent N duplicate warnings in configs that use N Pro resources.
//   - Once the floor advisory has been considered (emitted or determined not to be
//     applicable), further Configure calls with empty minVer skip the version fetch
//     entirely — there is nothing left to check on those code paths.
type Data struct {
	Client *jamfplatform.Client

	proMu      sync.Mutex
	proVersion string

	floorMu      sync.Mutex
	floorHandled bool

	// versionFetcher is the function used to retrieve the tenant Jamf Pro version.
	// Tests override this to avoid real HTTP calls. Nil means use the default SDK path.
	versionFetcher func(ctx context.Context) (string, error)
}

// New wraps a configured SDK client in a Data value.
func New(client *jamfplatform.Client) *Data {
	return &Data{Client: client}
}

// GetJamfProVersion fetches the tenant's Jamf Pro version. Successful results are
// cached for the lifetime of the Data value. Errors are not cached — the next call
// retries the fetch. Resources with empty minJamfProVersion should not call this —
// fetching only fires when a Pro resource with a version requirement is in the config.
func (d *Data) GetJamfProVersion(ctx context.Context) (string, error) {
	d.proMu.Lock()
	defer d.proMu.Unlock()
	if d.proVersion != "" {
		return d.proVersion, nil
	}
	fetch := d.versionFetcher
	if fetch == nil {
		fetch = d.defaultVersionFetch
	}
	v, err := fetch(ctx)
	if err != nil {
		return "", err
	}
	d.proVersion = v
	return v, nil
}

// defaultVersionFetch is the production version fetcher backed by the Jamf Pro SDK.
func (d *Data) defaultVersionFetch(ctx context.Context) (string, error) {
	v, err := pro.New(d.Client).GetJamfProVersionV1(ctx)
	if err != nil {
		return "", err
	}
	if v == nil {
		return "", nil
	}
	return v.Version, nil
}

// providerFloorWarning returns a warning diagnostic if d.proVersion is below
// ProviderMinJamfProVersion. Fires at most once per Data value: the first call
// records the consideration (whether or not a warning is emitted) and every later
// call returns nil so configs with many Pro resources do not surface duplicate
// warnings. The caller must have invoked GetJamfProVersion before calling this so
// d.proVersion is populated.
func (d *Data) providerFloorWarning() diag.Diagnostic {
	d.floorMu.Lock()
	defer d.floorMu.Unlock()
	if d.floorHandled {
		return nil
	}
	d.floorHandled = true
	if d.proVersion == "" {
		return nil
	}
	return helpers.WarnIfBelowProviderFloor(d.proVersion, ProviderMinJamfProVersion)
}

// floorAlreadyHandled reports whether the provider-floor advisory has already been
// considered for this Data value. Used by ConfigurePro to short-circuit the version
// fetch on resources with empty minVer once the floor has been emitted or skipped.
func (d *Data) floorAlreadyHandled() bool {
	d.floorMu.Lock()
	defer d.floorMu.Unlock()
	return d.floorHandled
}

// ConfigurePro is the shared Configure boilerplate for every Pro resource, data source,
// list resource, and action. It type-asserts providerData into *Data, fetches the Jamf
// Pro tenant version (lazily, cached on the Data value), runs the per-resource minimum
// version gate when minVer is non-empty, and surfaces the provider-floor advisory
// warning when the tenant is below the provider build target.
//
// Once the floor warning has been considered for the Data value, subsequent Configure
// calls with empty minVer skip the version fetch entirely — there is nothing left to
// evaluate on those code paths, so the network round-trip is avoided.
//
// resourceType is the fully-qualified Terraform type name used in diagnostic messages
// (e.g. "jamfplatform_pro_category"). Returns a *pro.Client ready for use; callers
// should check resp.Diagnostics.HasError() before using it.
//
// Returns (nil, nil) when providerData is nil (the framework calls Configure with a
// nil ProviderData during early lifecycle — that is not an error, the resource simply
// remains unconfigured until a later Configure call provides the data).
func ConfigurePro(ctx context.Context, providerData any, minVer, resourceType string) (*pro.Client, diag.Diagnostics) {
	var diags diag.Diagnostics
	if providerData == nil {
		return nil, diags
	}
	pd, ok := providerData.(*Data)
	if !ok {
		diags.AddError(
			"Unexpected Configure Type",
			fmt.Sprintf("Expected *providerdata.Data, got: %T. Please report this issue to the provider developers.", providerData),
		)
		return nil, diags
	}
	client := pro.New(pd.Client)

	// Fast path: empty per-resource minVer and the provider-floor advisory has already
	// been considered for this Data value → nothing left to fetch.
	if minVer == "" && pd.floorAlreadyHandled() {
		return client, diags
	}

	version, err := pd.GetJamfProVersion(ctx)
	if err != nil {
		if minVer == "" {
			return client, diags
		}
		diags.AddError(
			"Failed to read Jamf Pro tenant version",
			fmt.Sprintf("%s requires Jamf Pro; could not read version: %s", resourceType, err),
		)
		return nil, diags
	}
	if minVer != "" {
		diags.Append(helpers.RequireMinJamfProVersion(version, minVer, resourceType)...)
	}
	if warn := pd.providerFloorWarning(); warn != nil {
		diags.Append(warn)
	}
	return client, diags
}
