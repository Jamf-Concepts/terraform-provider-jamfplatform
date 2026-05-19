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
type Data struct {
	Client *jamfplatform.Client

	proVersionOnce sync.Once
	proVersion     string
	proVersionErr  error
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

// providerFloorWarning returns a warning diagnostic if d.proVersion is below
// ProviderMinJamfProVersion. The caller must have invoked GetJamfProVersion before
// calling this — that establishes the happens-before relationship needed to read
// d.proVersion without a race. Returns nil when at/above floor or when proVersion
// has not been populated.
func (d *Data) providerFloorWarning() diag.Diagnostic {
	if d.proVersion == "" {
		return nil
	}
	return helpers.WarnIfBelowProviderFloor(d.proVersion, ProviderMinJamfProVersion)
}

// ConfigurePro is the shared Configure boilerplate for every Pro resource, data source,
// list resource, and action. It type-asserts providerData into *Data, fetches the Jamf
// Pro tenant version (lazily, cached on the Data value), runs the per-resource minimum
// version gate when minVer is non-empty, and surfaces the provider-floor advisory
// warning when the tenant is below the provider build target.
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
