// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package patch_external_source

import (
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// portValueOrNull maps an SDK *int port onto a Terraform Int64. The classic
// /patchexternalsources wire emits an empty <port/> element for an unset port,
// which the XML decoder unmarshals to a non-nil pointer to 0 (NOT nil). Port is
// an Optional-only attribute, so a server-echoed 0 for an unconfigured port
// would otherwise drift against a null plan. We collapse both nil and 0 to null:
// 0 is never a valid definitions-server port, so it is safe to treat as unset.
//
// NOTE (untested transition): clearing a previously-set port (config 8443 →
// removed) relies on the classic PUT clearing the field server-side rather than
// merge-retaining it. We cannot probe that without a tenant; the required acc
// test changes the port rather than clearing it, so this path is unverified.
// Watch it on `make testacc`.
func portValueOrNull(p *int) types.Int64 {
	if p != nil && *p != 0 {
		return types.Int64Value(int64(*p))
	}
	return types.Int64Null()
}

// assignPatchExternalSourceResourceModel populates a resource model from a
// PatchExternalSource response. state.ID is only overwritten when the API ID is
// non-nil so a transient GET that drops the ID does not clobber the ID already
// persisted from Create. Optional+Computed bools are assigned directly from the
// server (it always echoes them) so the Computed slot surfaces the server
// default.
func assignPatchExternalSourceResourceModel(state *PatchExternalSourceResourceModel, s *proclassic.PatchExternalSource) {
	if s == nil {
		return
	}
	if s.ID != nil {
		state.ID = helpers.StringValueFromIntPtr(s.ID)
	}
	if s.Name != nil {
		state.Name = helpers.StringPointerValueOrNull(s.Name)
	}
	state.Enabled = helpers.BoolPointerValueOrNull(s.Enabled)
	state.HostName = helpers.StringPointerValueOrNull(s.HostName)
	state.Port = portValueOrNull(s.Port)
	state.SslEnabled = helpers.BoolPointerValueOrNull(s.SslEnabled)
	state.CertificateValidationEnabled = helpers.BoolPointerValueOrNull(s.CertificateValidationEnabled)
}

// assignPatchExternalSourceDataSourceModel populates a data source model from a
// PatchExternalSource response. Symmetric with the resource assignment: the ID
// and Name selectors are preserved when the API field is nil so the
// caller-supplied lookup value is not silently nulled. All remaining attributes
// are surfaced from the response.
func assignPatchExternalSourceDataSourceModel(state *PatchExternalSourceDataSourceModel, s *proclassic.PatchExternalSource) {
	if s == nil {
		return
	}
	if s.ID != nil {
		state.ID = helpers.StringValueFromIntPtr(s.ID)
	}
	if s.Name != nil {
		state.Name = helpers.StringPointerValueOrNull(s.Name)
	}
	state.Enabled = helpers.BoolPointerValueOrNull(s.Enabled)
	state.HostName = helpers.StringPointerValueOrNull(s.HostName)
	state.Port = portValueOrNull(s.Port)
	state.SslEnabled = helpers.BoolPointerValueOrNull(s.SslEnabled)
	state.CertificateValidationEnabled = helpers.BoolPointerValueOrNull(s.CertificateValidationEnabled)
}
