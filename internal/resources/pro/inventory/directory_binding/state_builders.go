// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package directory_binding

import (
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// assignDirectoryBindingResourceModel populates a resource model from a
// DirectoryBinding response. state.ID is only overwritten when the API ID is
// non-nil so a transient GET that drops the ID does not clobber the value
// persisted from Create.
//
// state.Password is deliberately NOT touched. The classic GET response never
// echoes the plaintext password — only `password_sha256`. Overwriting
// state.Password from the response would null it on every refresh, surfacing
// as drift on the next plan and asking the user to re-supply the password.
// The user's plaintext stays in state from the last apply; the server hash
// surfaces separately as state.PasswordSha256 (Computed) so out-of-band
// hash changes are still observable.
//
// The empty `<powerbroker_identity_services/>` wire element does not surface
// in state — the schema exposes no nested block for PowerBroker since it
// carries no per-type fields. The `type` field on its own conveys the
// PowerBroker identity.
func assignDirectoryBindingResourceModel(state *DirectoryBindingResourceModel, b *proclassic.DirectoryBinding) diag.Diagnostics {
	var diags diag.Diagnostics
	if b == nil {
		return diags
	}
	if b.ID != nil {
		state.ID = helpers.StringValueFromIntPtr(b.ID)
	}
	state.Name = helpers.StringPointerValueOrNull(b.Name)
	state.Type = helpers.StringPointerValueOrNull(b.Type)
	state.Domain = helpers.StringPointerValueOrNull(b.Domain)
	state.Username = helpers.StringPointerValueOrNull(b.Username)
	state.PasswordSha256 = helpers.StringPointerValueOrNull(b.PasswordSha256)
	state.ComputerOU = helpers.StringPointerValueOrNull(b.ComputerOu)
	state.Priority = int64ValueFromPtr(b.Priority)

	state.ActiveDirectory = assignActiveDirectoryModel(b.ActiveDirectory)
	state.OpenDirectory = assignOpenDirectoryModel(b.OpenDirectory)
	state.Admitmac = assignAdmitmacModel(b.Admitmac)
	state.Centrify = assignCentrifyModel(b.Centrify)

	return diags
}

// assignDirectoryBindingDataSourceModel populates a data source model from a
// DirectoryBinding response. Symmetric with the resource builder, minus the
// `password` field (data source is read-only — no user-supplied plaintext to
// preserve).
func assignDirectoryBindingDataSourceModel(state *DirectoryBindingDataSourceModel, b *proclassic.DirectoryBinding) diag.Diagnostics {
	var diags diag.Diagnostics
	if b == nil {
		return diags
	}
	if b.ID != nil {
		state.ID = helpers.StringValueFromIntPtr(b.ID)
	}
	state.Name = helpers.StringPointerValueOrNull(b.Name)
	state.Type = helpers.StringPointerValueOrNull(b.Type)
	state.Domain = helpers.StringPointerValueOrNull(b.Domain)
	state.Username = helpers.StringPointerValueOrNull(b.Username)
	state.PasswordSha256 = helpers.StringPointerValueOrNull(b.PasswordSha256)
	state.ComputerOU = helpers.StringPointerValueOrNull(b.ComputerOu)
	state.Priority = int64ValueFromPtr(b.Priority)

	state.ActiveDirectory = assignActiveDirectoryModel(b.ActiveDirectory)
	state.OpenDirectory = assignOpenDirectoryModel(b.OpenDirectory)
	state.Admitmac = assignAdmitmacModel(b.Admitmac)
	state.Centrify = assignCentrifyModel(b.Centrify)

	return diags
}

// assignActiveDirectoryModel decodes the nested SDK block into the TF
// model, or returns nil when the API did not include the block. A nil
// return tells the framework to omit the SingleNestedAttribute from state
// entirely rather than surfacing it as a struct full of nulls.
func assignActiveDirectoryModel(a *proclassic.DirectoryBindingActiveDirectory) *directoryBindingActiveDirectoryModel {
	if a == nil {
		return nil
	}
	return &directoryBindingActiveDirectoryModel{
		Forest:                  helpers.StringPointerValueOrNull(a.Forest),
		CreateMobileAccount:     helpers.BoolPointerValueOrNull(a.CacheLastUser),
		RequireConfirmation:     helpers.BoolPointerValueOrNull(a.RequireConfirmation),
		ForceLocalHomeDirectory: helpers.BoolPointerValueOrNull(a.LocalHome),
		UseUncPath:              helpers.BoolPointerValueOrNull(a.UseUncPath),
		NetworkProtocol:         helpers.StringPointerValueOrNull(a.MountStyle),
		DefaultShell:            helpers.StringPointerValueOrNull(a.DefaultShell),
		UIDAttributeMapping:     helpers.StringPointerValueOrNull(a.Uid),
		UserGIDAttributeMapping: helpers.StringPointerValueOrNull(a.UserGid),
		GIDAttributeMapping:     helpers.StringPointerValueOrNull(a.Gid),
		MultipleDomains:         helpers.BoolPointerValueOrNull(a.MultipleDomains),
		PreferredDomain:         helpers.StringPointerValueOrNull(a.PreferredDomain),
		AdminGroups:             helpers.StringPointerValueOrNull(a.AdminGroups),
	}
}

// assignOpenDirectoryModel decodes the nested SDK block into the TF model.
func assignOpenDirectoryModel(o *proclassic.DirectoryBindingOpenDirectory) *directoryBindingOpenDirectoryModel {
	if o == nil {
		return nil
	}
	return &directoryBindingOpenDirectoryModel{
		EncryptUsingSSL:      helpers.BoolPointerValueOrNull(o.EncryptUsingSsl),
		PerformSecureBind:    helpers.BoolPointerValueOrNull(o.PerformSecureBind),
		UseForAuthentication: helpers.BoolPointerValueOrNull(o.UseForAuthentication),
		UseForContacts:       helpers.BoolPointerValueOrNull(o.UseForContacts),
	}
}

// assignAdmitmacModel decodes the nested SDK block into the TF model.
func assignAdmitmacModel(a *proclassic.DirectoryBindingAdmitmac) *directoryBindingAdmitmacModel {
	if a == nil {
		return nil
	}
	return &directoryBindingAdmitmacModel{
		RequireConfirmation:     helpers.BoolPointerValueOrNull(a.RequireConfirmation),
		HomeLocation:            helpers.StringPointerValueOrNull(a.LocalHome),
		NetworkProtocol:         helpers.StringPointerValueOrNull(a.MountStyle),
		DefaultShell:            helpers.StringPointerValueOrNull(a.DefaultShell),
		MountNetworkHome:        helpers.BoolPointerValueOrNull(a.MountNetworkHome),
		PlaceHomeFolders:        helpers.StringPointerValueOrNull(a.PlaceHomeFolders),
		UIDAttributeMapping:     helpers.StringPointerValueOrNull(a.Uid),
		UserGIDAttributeMapping: helpers.StringPointerValueOrNull(a.UserGid),
		GIDAttributeMapping:     helpers.StringPointerValueOrNull(a.Gid),
		AdminGroup:              helpers.StringPointerValueOrNull(a.AdminGroup),
		CachedCredentials:       int64ValueFromPtr(a.CachedCredentials),
		AddUserToLocal:          helpers.BoolPointerValueOrNull(a.AddUserToLocal),
		UsersOU:                 helpers.StringPointerValueOrNull(a.UsersOu),
		GroupsOU:                helpers.StringPointerValueOrNull(a.GroupsOu),
		PrintersOU:              helpers.StringPointerValueOrNull(a.PrintersOu),
		SharedFoldersOU:         helpers.StringPointerValueOrNull(a.SharedFoldersOu),
	}
}

// assignCentrifyModel decodes the nested SDK block into the TF model.
func assignCentrifyModel(c *proclassic.DirectoryBindingCentrify) *directoryBindingCentrifyModel {
	if c == nil {
		return nil
	}
	return &directoryBindingCentrifyModel{
		WorkstationMode:       helpers.BoolPointerValueOrNull(c.WorkstationMode),
		OverwriteExisting:     helpers.BoolPointerValueOrNull(c.OverwriteExisting),
		UpdatePAM:             helpers.BoolPointerValueOrNull(c.UpdatePAM),
		Zone:                  helpers.StringPointerValueOrNull(c.Zone),
		PreferredDomainServer: helpers.StringPointerValueOrNull(c.PreferredDomainServer),
	}
}

// int64ValueFromPtr converts a nil-safe *int (the SDK shape) into a TF
// Int64, mapping nil to null. The SDK uses *int for Priority and
// CachedCredentials.
func int64ValueFromPtr(p *int) types.Int64 {
	if p == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(*p))
}
