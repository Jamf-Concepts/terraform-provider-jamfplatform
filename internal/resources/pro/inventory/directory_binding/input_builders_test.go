// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package directory_binding

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestBuildDirectoryBindingInput_ActiveDirectory covers the AD path: the
// flat envelope and the active_directory nested block both reach the SDK
// payload; the other three per-type blocks stay nil so the wire encoder
// omits them.
func TestBuildDirectoryBindingInput_ActiveDirectory(t *testing.T) {
	plan := DirectoryBindingResourceModel{
		Name:       types.StringValue("ad-prod"),
		Priority:   types.Int64Value(1),
		Type:       types.StringValue(typeActiveDirectory),
		Domain:     types.StringValue("corp.example.com"),
		Username:   types.StringValue("joiner"),
		Password:   types.StringValue("hunter2"),
		ComputerOU: types.StringValue("OU=Macs,DC=corp,DC=example,DC=com"),
		ActiveDirectory: &directoryBindingActiveDirectoryModel{
			Forest:                  types.StringValue("corp.example.com"),
			CreateMobileAccount:     types.BoolValue(true),
			RequireConfirmation:     types.BoolValue(true),
			ForceLocalHomeDirectory: types.BoolValue(true),
			UseUncPath:              types.BoolValue(true),
			NetworkProtocol:         types.StringValue("smb"),
			DefaultShell:            types.StringValue("/bin/bash"),
			UIDAttributeMapping:     types.StringValue("uid-attr"),
			UserGIDAttributeMapping: types.StringValue("gid-attr"),
			GIDAttributeMapping:     types.StringValue("gid-attr"),
			MultipleDomains:         types.BoolValue(false),
			PreferredDomain:         types.StringValue("dc01.corp.example.com"),
			AdminGroups:             types.StringValue("Mac Admins,Domain Admins"),
		},
	}

	got := buildDirectoryBindingInput(plan)

	if got.Name == nil || *got.Name != "ad-prod" {
		t.Errorf("expected Name=ad-prod, got %v", got.Name)
	}
	if got.Priority == nil || *got.Priority != 1 {
		t.Errorf("expected Priority=1, got %v", got.Priority)
	}
	if got.Type == nil || *got.Type != "Active Directory" {
		t.Errorf("expected Type=Active Directory, got %v", got.Type)
	}
	if got.Password == nil || *got.Password != "hunter2" {
		t.Errorf("expected Password to round-trip, got %v", got.Password)
	}
	if got.ActiveDirectory == nil {
		t.Fatalf("expected ActiveDirectory nested struct, got nil")
	}
	if got.ActiveDirectory.AdminGroups == nil || *got.ActiveDirectory.AdminGroups != "Mac Admins,Domain Admins" {
		t.Errorf("AD admin groups did not round-trip: %v", got.ActiveDirectory.AdminGroups)
	}
	if got.OpenDirectory != nil || got.Admitmac != nil || got.Centrify != nil || got.PowerbrokerIdentityServices != nil {
		t.Errorf("only ActiveDirectory should be populated; other type blocks must stay nil")
	}
}

// TestBuildDirectoryBindingInput_OpenDirectory covers the Open Directory
// path with the wire type value "Open Directory" (not the UI label "Apple
// Open Directory").
func TestBuildDirectoryBindingInput_OpenDirectory(t *testing.T) {
	plan := DirectoryBindingResourceModel{
		Name: types.StringValue("od"),
		Type: types.StringValue(typeOpenDirectory),
		OpenDirectory: &directoryBindingOpenDirectoryModel{
			EncryptUsingSSL:      types.BoolValue(true),
			PerformSecureBind:    types.BoolValue(true),
			UseForAuthentication: types.BoolValue(true),
			UseForContacts:       types.BoolValue(false),
		},
	}
	got := buildDirectoryBindingInput(plan)

	if got.Type == nil || *got.Type != "Open Directory" {
		t.Errorf("expected Type wire value 'Open Directory', got %v", got.Type)
	}
	if got.OpenDirectory == nil {
		t.Fatalf("expected OpenDirectory nested struct, got nil")
	}
	if got.OpenDirectory.EncryptUsingSsl == nil || !*got.OpenDirectory.EncryptUsingSsl {
		t.Errorf("EncryptUsingSsl did not round-trip")
	}
	if got.OpenDirectory.UseForContacts == nil || *got.OpenDirectory.UseForContacts {
		t.Errorf("UseForContacts=false did not round-trip")
	}
	if got.ActiveDirectory != nil || got.Admitmac != nil || got.Centrify != nil || got.PowerbrokerIdentityServices != nil {
		t.Errorf("only OpenDirectory should be populated")
	}
}

// TestBuildDirectoryBindingInput_PowerBroker covers the empty-block path.
// PowerBroker carries no per-type fields; the input builder must synthesise
// a non-nil empty SDK struct so the wire encoder emits
// `<powerbroker_identity_services/>`.
func TestBuildDirectoryBindingInput_PowerBroker(t *testing.T) {
	plan := DirectoryBindingResourceModel{
		Name: types.StringValue("pb"),
		Type: types.StringValue(typePowerBroker),
	}
	got := buildDirectoryBindingInput(plan)

	if got.PowerbrokerIdentityServices == nil {
		t.Fatalf("PowerBroker type must synthesise an empty PowerbrokerIdentityServices struct so the empty wire element round-trips")
	}
	if got.ActiveDirectory != nil || got.OpenDirectory != nil || got.Admitmac != nil || got.Centrify != nil {
		t.Errorf("only PowerbrokerIdentityServices should be populated")
	}
}

// TestBuildDirectoryBindingInput_ADmitMac covers the ADmitMac path.
// Includes the *int CachedCredentials field and confirms `local_home` is
// preserved as a string (not coerced into a bool — distinct from the AD
// field of the same name).
func TestBuildDirectoryBindingInput_ADmitMac(t *testing.T) {
	plan := DirectoryBindingResourceModel{
		Name: types.StringValue("admitmac"),
		Type: types.StringValue(typeADmitMac),
		Admitmac: &directoryBindingAdmitmacModel{
			RequireConfirmation:     types.BoolValue(false),
			HomeLocation:            types.StringValue("Local"),
			NetworkProtocol:         types.StringValue("smb"),
			DefaultShell:            types.StringValue("/bin/bash"),
			MountNetworkHome:        types.BoolValue(false),
			PlaceHomeFolders:        types.StringValue("/Users"),
			UIDAttributeMapping:     types.StringValue("uid"),
			UserGIDAttributeMapping: types.StringValue("ugid"),
			GIDAttributeMapping:     types.StringValue("gid"),
			AdminGroup:              types.StringValue("Mac Admins"),
			CachedCredentials:       types.Int64Value(10),
			AddUserToLocal:          types.BoolValue(true),
			UsersOU:                 types.StringValue("OU=Users,DC=corp,DC=example,DC=com"),
			GroupsOU:                types.StringValue("OU=Groups,DC=corp,DC=example,DC=com"),
			PrintersOU:              types.StringValue("OU=Printers,DC=corp,DC=example,DC=com"),
			SharedFoldersOU:         types.StringValue("OU=Shares,DC=corp,DC=example,DC=com"),
		},
	}
	got := buildDirectoryBindingInput(plan)

	if got.Admitmac == nil {
		t.Fatalf("expected Admitmac nested struct, got nil")
	}
	if got.Admitmac.LocalHome == nil || *got.Admitmac.LocalHome != "Local" {
		t.Errorf("ADmitMac LocalHome must round-trip as string 'Local', got %v", got.Admitmac.LocalHome)
	}
	if got.Admitmac.CachedCredentials == nil || *got.Admitmac.CachedCredentials != 10 {
		t.Errorf("CachedCredentials must round-trip as *int 10, got %v", got.Admitmac.CachedCredentials)
	}
	if got.ActiveDirectory != nil || got.OpenDirectory != nil || got.Centrify != nil || got.PowerbrokerIdentityServices != nil {
		t.Errorf("only Admitmac should be populated")
	}
}

// TestBuildDirectoryBindingInput_Centrify covers the Centrify path and
// confirms UpdatePAM (uppercase wire element <update_PAM>) round-trips.
func TestBuildDirectoryBindingInput_Centrify(t *testing.T) {
	plan := DirectoryBindingResourceModel{
		Name: types.StringValue("centrify"),
		Type: types.StringValue(typeCentrify),
		Centrify: &directoryBindingCentrifyModel{
			WorkstationMode:       types.BoolValue(false),
			OverwriteExisting:     types.BoolValue(true),
			UpdatePAM:             types.BoolValue(true),
			Zone:                  types.StringValue("macs"),
			PreferredDomainServer: types.StringValue("dc01.corp.example.com"),
		},
	}
	got := buildDirectoryBindingInput(plan)

	if got.Centrify == nil {
		t.Fatalf("expected Centrify nested struct, got nil")
	}
	if got.Centrify.UpdatePAM == nil || !*got.Centrify.UpdatePAM {
		t.Errorf("UpdatePAM must round-trip; got %v", got.Centrify.UpdatePAM)
	}
	if got.ActiveDirectory != nil || got.OpenDirectory != nil || got.Admitmac != nil || got.PowerbrokerIdentityServices != nil {
		t.Errorf("only Centrify should be populated")
	}
}

// TestBuildDirectoryBindingInput_PasswordOmittedWhenNull verifies that a
// null `password` produces a nil *string so the wire field is omitted.
// Classic's partial-merge semantics treat the omitted field as
// "preserve" — exactly the behaviour we want for write-only credentials.
func TestBuildDirectoryBindingInput_PasswordOmittedWhenNull(t *testing.T) {
	plan := DirectoryBindingResourceModel{
		Name:     types.StringValue("ad"),
		Type:     types.StringValue(typeActiveDirectory),
		Password: types.StringNull(),
	}
	got := buildDirectoryBindingInput(plan)

	if got.Password != nil {
		t.Errorf("null Password must serialise to nil, got %v", *got.Password)
	}
}

// TestBuildDirectoryBindingInput_IDNotEmittedOnWrite verifies the wire
// payload never carries an ID — Create uses path id="0" and Update derives
// the ID from state. Letting the body carry an ID would race with the URL
// path and the server rejects the combination.
func TestBuildDirectoryBindingInput_IDNotEmittedOnWrite(t *testing.T) {
	plan := DirectoryBindingResourceModel{
		ID:   types.StringValue("123"),
		Name: types.StringValue("ad"),
		Type: types.StringValue(typeActiveDirectory),
	}
	got := buildDirectoryBindingInput(plan)

	if got.ID != nil {
		t.Errorf("ID must not appear on write payload (Create uses path id=\"0\"; Update derives from state)")
	}
}

// TestBuildDirectoryBindingInput_TypeMismatchedBlockOmitsBlock verifies
// that when the user supplies a block whose name does not match `type`,
// the input builder still keys off `type` and emits only the matching
// nested block — the validator catches this combination at plan time
// already, but the input builder must not double-emit if the validator
// fails to trigger (defence-in-depth).
func TestBuildDirectoryBindingInput_TypeMismatchedBlockOmitsBlock(t *testing.T) {
	plan := DirectoryBindingResourceModel{
		Name: types.StringValue("ad"),
		Type: types.StringValue(typeActiveDirectory),
		Centrify: &directoryBindingCentrifyModel{
			Zone: types.StringValue("macs"),
		},
	}
	got := buildDirectoryBindingInput(plan)

	if got.Centrify != nil {
		t.Errorf("input builder must key off Type, not user-supplied blocks; Centrify block must not reach the wire when Type=Active Directory")
	}
}
