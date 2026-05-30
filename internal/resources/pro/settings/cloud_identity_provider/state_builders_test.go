// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package cloud_identity_provider

import (
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// --- assignGoogleState -------------------------------------------------------

// TestAssignGoogleState_MapsServerFields verifies that all scalar server fields
// are folded correctly into the resource model from a GET response.
func TestAssignGoogleState_MapsServerFields(t *testing.T) {
	state := CloudIdentityProviderResourceModel{}
	resp := &pro.LdapConfigurationResponse{
		CloudIDPCommon: &pro.CloudIDPCommon{
			ID:           "ldap-001",
			DisplayName:  "My Google LDAP",
			ProviderName: providerGoogle,
		},
		Server: &pro.CloudLdapServerResponse{
			ServerURL:                                "ldap.google.com",
			DomainName:                               "example.com",
			Port:                                     636,
			ConnectionType:                           "LDAPS",
			ConnectionTimeout:                        15,
			SearchTimeout:                            60,
			UseWildcards:                             true,
			Enabled:                                  true,
			MembershipCalculationOptimizationEnabled: false,
			Keystore: &pro.CloudLdapKeystore{
				FileName: "google-ldap.p12",
				Type:     "PKCS12",
				Subject:  "CN=Jamf Cloud LDAP Client",
			},
		},
	}

	assignGoogleState(&state, resp, types.Int64Value(1))

	if state.ID.ValueString() != "ldap-001" {
		t.Errorf("ID mismatch: got %q, want %q", state.ID.ValueString(), "ldap-001")
	}
	if state.DisplayName.ValueString() != "My Google LDAP" {
		t.Errorf("DisplayName mismatch")
	}
	if state.ProviderName.ValueString() != providerGoogle {
		t.Errorf("ProviderName mismatch")
	}
	if state.Google == nil || state.Google.Server == nil {
		t.Fatalf("Google.Server must not be nil")
	}
	s := state.Google.Server
	if s.ServerURL.ValueString() != "ldap.google.com" {
		t.Errorf("ServerURL mismatch: got %q", s.ServerURL.ValueString())
	}
	if s.DomainName.ValueString() != "example.com" {
		t.Errorf("DomainName mismatch")
	}
	if s.Port.ValueInt64() != 636 {
		t.Errorf("Port mismatch: got %d", s.Port.ValueInt64())
	}
	if s.ConnectionType.ValueString() != "LDAPS" {
		t.Errorf("ConnectionType mismatch")
	}
	if !s.UseWildcards.ValueBool() {
		t.Errorf("UseWildcards mismatch")
	}
	if !s.Enabled.ValueBool() {
		t.Errorf("Enabled mismatch")
	}
}

// TestAssignGoogleState_KeystoreEchoesPopulated verifies that the server-derived
// keystore echoes (type, subject, expiration_date, file_name) are populated from
// the GET response.
func TestAssignGoogleState_KeystoreEchoesPopulated(t *testing.T) {
	exp := "2027-12-31T00:00:00"
	state := CloudIdentityProviderResourceModel{}
	resp := &pro.LdapConfigurationResponse{
		CloudIDPCommon: &pro.CloudIDPCommon{ID: "x"},
		Server: &pro.CloudLdapServerResponse{
			Keystore: &pro.CloudLdapKeystore{
				FileName:       "cert.p12",
				Type:           "PKCS12",
				Subject:        "CN=Test",
				ExpirationDate: &exp,
			},
		},
	}

	assignGoogleState(&state, resp, types.Int64Null())

	ks := state.Google.Server.Keystore
	if ks == nil {
		t.Fatalf("Keystore must not be nil")
	}
	if ks.FileName.ValueString() != "cert.p12" {
		t.Errorf("FileName mismatch: got %q", ks.FileName.ValueString())
	}
	if ks.Type.ValueString() != "PKCS12" {
		t.Errorf("Type mismatch")
	}
	if ks.Subject.ValueString() != "CN=Test" {
		t.Errorf("Subject mismatch")
	}
	if ks.ExpirationDate.ValueString() != "2027-12-31T00:00:00" {
		t.Errorf("ExpirationDate mismatch: got %q", ks.ExpirationDate.ValueString())
	}
}

// TestAssignGoogleState_KeystoreFileAndPasswordStayNull verifies that the
// WriteOnly keystore.file and keystore.password attributes remain null in
// state — they must never be populated from a server response.
func TestAssignGoogleState_KeystoreFileAndPasswordStayNull(t *testing.T) {
	state := CloudIdentityProviderResourceModel{}
	resp := &pro.LdapConfigurationResponse{
		CloudIDPCommon: &pro.CloudIDPCommon{ID: "y"},
		Server: &pro.CloudLdapServerResponse{
			Keystore: &pro.CloudLdapKeystore{FileName: "cert.p12"},
		},
	}

	assignGoogleState(&state, resp, types.Int64Value(2))

	ks := state.Google.Server.Keystore
	if ks == nil {
		t.Fatal("Keystore must not be nil")
	}
	if !ks.File.IsNull() {
		t.Errorf("keystore.file must stay null in state (WriteOnly); got %q", ks.File.ValueString())
	}
	if !ks.Password.IsNull() {
		t.Errorf("keystore.password must stay null in state (WriteOnly); got %q", ks.Password.ValueString())
	}
}

// TestAssignGoogleState_WoVersionPreserved verifies that the prior wo_version
// value is threaded into the new state so a refresh does not null it out.
func TestAssignGoogleState_WoVersionPreserved(t *testing.T) {
	state := CloudIdentityProviderResourceModel{}
	resp := &pro.LdapConfigurationResponse{
		CloudIDPCommon: &pro.CloudIDPCommon{ID: "z"},
		Server: &pro.CloudLdapServerResponse{
			Keystore: &pro.CloudLdapKeystore{FileName: "cert.p12"},
		},
	}

	assignGoogleState(&state, resp, types.Int64Value(3))

	ks := state.Google.Server.Keystore
	if ks.WoVersion.ValueInt64() != 3 {
		t.Errorf("wo_version must be preserved from priorWoVersion; got %d", ks.WoVersion.ValueInt64())
	}
}

// TestAssignGoogleState_WoVersionPreservedNull verifies that a null prior
// wo_version remains null in state after assignGoogleState.
func TestAssignGoogleState_WoVersionPreservedNull(t *testing.T) {
	state := CloudIdentityProviderResourceModel{}
	resp := &pro.LdapConfigurationResponse{
		CloudIDPCommon: &pro.CloudIDPCommon{ID: "z2"},
		Server: &pro.CloudLdapServerResponse{
			Keystore: &pro.CloudLdapKeystore{FileName: "cert.p12"},
		},
	}

	assignGoogleState(&state, resp, types.Int64Null())

	ks := state.Google.Server.Keystore
	if !ks.WoVersion.IsNull() {
		t.Errorf("null prior wo_version must remain null; got %d", ks.WoVersion.ValueInt64())
	}
}

// TestAssignGoogleState_ExpirationDateNilMapsToNull verifies that a nil
// ExpirationDate pointer in the server response maps to a null TF string (not
// an empty string).
func TestAssignGoogleState_ExpirationDateNilMapsToNull(t *testing.T) {
	state := CloudIdentityProviderResourceModel{}
	resp := &pro.LdapConfigurationResponse{
		CloudIDPCommon: &pro.CloudIDPCommon{ID: "exp-nil"},
		Server: &pro.CloudLdapServerResponse{
			Keystore: &pro.CloudLdapKeystore{
				FileName:       "cert.p12",
				ExpirationDate: nil, // no expiration date
			},
		},
	}

	assignGoogleState(&state, resp, types.Int64Null())

	ks := state.Google.Server.Keystore
	if !ks.ExpirationDate.IsNull() {
		t.Errorf("nil ExpirationDate must map to null; got %q", ks.ExpirationDate.ValueString())
	}
}

// TestAssignGoogleState_NilSafe verifies that a nil response does not panic and
// leaves state untouched.
func TestAssignGoogleState_NilSafe(t *testing.T) {
	state := CloudIdentityProviderResourceModel{ID: types.StringValue("existing")}
	assignGoogleState(&state, nil, types.Int64Null())
	if state.ID.ValueString() != "existing" {
		t.Errorf("nil response must leave state untouched; got %q", state.ID.ValueString())
	}
}

// TestAssignGoogleState_MappingsPopulated verifies that mapping fields are
// copied into the model when the GET response includes them AND the user
// authored those sub-blocks. mappings is Optional (not Computed), so the prior
// model must carry the sub-blocks for them to be surfaced (otherwise a
// server-generated block the user did not configure would trip a "planned
// null, got object" consistency error).
func TestAssignGoogleState_MappingsPopulated(t *testing.T) {
	state := CloudIdentityProviderResourceModel{
		Google: &cloudIdentityProviderGoogleModel{
			Mappings: &cloudLdapMappingsModel{
				UserMappings:       &cloudLdapUserMappingsModel{},
				GroupMappings:      &cloudLdapGroupMappingsModel{},
				MembershipMappings: &cloudLdapMembershipMappingsModel{},
			},
		},
	}
	resp := &pro.LdapConfigurationResponse{
		CloudIDPCommon: &pro.CloudIDPCommon{ID: "maps-001"},
		Server: &pro.CloudLdapServerResponse{
			Keystore: &pro.CloudLdapKeystore{FileName: "cert.p12"},
		},
		Mappings: &pro.CloudLdapMappingsResponse{
			UserMappings: &pro.UserMappings{
				ObjectClassLimitation: "ANY_OBJECT_CLASSES",
				UserID:                "uid",
				Username:              "sAMAccountName",
			},
			GroupMappings: &pro.GroupMappings{
				GroupName: "cn",
			},
			MembershipMappings: &pro.MembershipMappings{
				GroupMembershipMapping: "memberOf",
			},
		},
	}

	assignGoogleState(&state, resp, types.Int64Null())

	if state.Google.Mappings == nil {
		t.Fatalf("Mappings must not be nil")
	}
	if state.Google.Mappings.UserMappings == nil {
		t.Fatal("UserMappings must not be nil")
	}
	if state.Google.Mappings.UserMappings.UserID.ValueString() != "uid" {
		t.Errorf("UserID mismatch: got %q", state.Google.Mappings.UserMappings.UserID.ValueString())
	}
	if state.Google.Mappings.GroupMappings == nil {
		t.Fatal("GroupMappings must not be nil")
	}
	if state.Google.Mappings.GroupMappings.GroupName.ValueString() != "cn" {
		t.Errorf("GroupName mismatch: got %q", state.Google.Mappings.GroupMappings.GroupName.ValueString())
	}
	if state.Google.Mappings.MembershipMappings == nil {
		t.Fatal("MembershipMappings must not be nil")
	}
	if state.Google.Mappings.MembershipMappings.GroupMembershipMapping.ValueString() != "memberOf" {
		t.Errorf("GroupMembershipMapping mismatch")
	}
}

// TestAssignGoogleState_MappingsNilSafe verifies assignMappingsState returns nil
// when the server returns no mappings block, and when the user did not author
// mappings (prior nil) even if the server returned a block.
func TestAssignGoogleState_MappingsNilSafe(t *testing.T) {
	if got := assignMappingsState(nil, nil); got != nil {
		t.Errorf("assignMappingsState(nil, nil) must return nil; got %+v", got)
	}
	// Server returned mappings, but the user never authored the block (prior
	// nil) — must stay nil to avoid a "planned null, got object" error.
	serverMappings := &pro.CloudLdapMappingsResponse{UserMappings: &pro.UserMappings{UserID: "uid"}}
	if got := assignMappingsState(serverMappings, nil); got != nil {
		t.Errorf("assignMappingsState with nil prior must return nil; got %+v", got)
	}
}

// --- assignAzureState --------------------------------------------------------

// TestAssignAzureState_MapsFields verifies all scalar Azure fields are folded
// into the resource model from a GET response. The server response carries the
// wire value "AZURE" (wireProviderAzure); assignAzureState must map it to the
// TF-facing value "ENTRA_ID" (providerEntraID) via providerNameFromWire.
func TestAssignAzureState_MapsFields(t *testing.T) {
	state := CloudIdentityProviderResourceModel{}
	resp := &pro.AzureConfiguration{
		CloudIDPCommon: &pro.CloudIDPCommon{
			ID:           "azure-001",
			DisplayName:  "My Entra ID",
			ProviderName: wireProviderAzure, // server sends "AZURE"
		},
		Server: &pro.AzureServerConfiguration{
			TenantID:                                 "tenant-uuid",
			SearchTimeout:                            30,
			Enabled:                                  true,
			MembershipCalculationOptimizationEnabled: false,
			TransitiveMembershipEnabled:              false,
			TransitiveMembershipUserField:            "userPrincipalName",
			TransitiveDirectoryMembershipEnabled:     false,
			Type:                                     "PUBLIC",
			Migrated:                                 false,
			DeprecatedConsent:                        false,
		},
	}

	assignAzureState(&state, resp)

	if state.ID.ValueString() != "azure-001" {
		t.Errorf("ID mismatch: got %q", state.ID.ValueString())
	}
	if state.DisplayName.ValueString() != "My Entra ID" {
		t.Errorf("DisplayName mismatch")
	}
	// Wire value "AZURE" must be mapped to TF value "ENTRA_ID".
	if state.ProviderName.ValueString() != providerEntraID {
		t.Errorf("ProviderName: wire AZURE must map to TF ENTRA_ID; got %q", state.ProviderName.ValueString())
	}
	if state.Azure == nil {
		t.Fatalf("Azure model must not be nil")
	}
	az := state.Azure
	if az.TenantID.ValueString() != "tenant-uuid" {
		t.Errorf("TenantID mismatch: got %q", az.TenantID.ValueString())
	}
	if az.SearchTimeout.ValueInt64() != 30 {
		t.Errorf("SearchTimeout mismatch: got %d", az.SearchTimeout.ValueInt64())
	}
	if !az.Enabled.ValueBool() {
		t.Errorf("Enabled mismatch")
	}
	if az.TransitiveMembershipUserField.ValueString() != "userPrincipalName" {
		t.Errorf("TransitiveMembershipUserField mismatch")
	}
}

// TestAssignAzureState_EchoesPopulated verifies the server-derived echo
// attributes (type, migrated, deprecated_consent) are populated.
func TestAssignAzureState_EchoesPopulated(t *testing.T) {
	state := CloudIdentityProviderResourceModel{}
	resp := &pro.AzureConfiguration{
		CloudIDPCommon: &pro.CloudIDPCommon{ID: "az-echoes"},
		Server: &pro.AzureServerConfiguration{
			Type:              "PUBLIC",
			Migrated:          true,
			DeprecatedConsent: true,
		},
	}

	assignAzureState(&state, resp)

	az := state.Azure
	if az.Type.ValueString() != "PUBLIC" {
		t.Errorf("Type echo mismatch: got %q", az.Type.ValueString())
	}
	if !az.Migrated.ValueBool() {
		t.Errorf("Migrated echo mismatch")
	}
	if !az.DeprecatedConsent.ValueBool() {
		t.Errorf("DeprecatedConsent echo mismatch")
	}
}

// TestAssignAzureState_MappingsNilSafe verifies that a nil Mappings pointer on
// the server response does not panic and leaves state.Azure.Mappings as nil.
func TestAssignAzureState_MappingsNilSafe(t *testing.T) {
	state := CloudIdentityProviderResourceModel{}
	resp := &pro.AzureConfiguration{
		CloudIDPCommon: &pro.CloudIDPCommon{ID: "az-no-maps"},
		Server: &pro.AzureServerConfiguration{
			TenantID: "t",
			Mappings: nil,
		},
	}

	assignAzureState(&state, resp)

	if state.Azure == nil {
		t.Fatal("Azure must not be nil")
	}
	if state.Azure.Mappings != nil {
		t.Errorf("Mappings must be nil when server returns nil; got %+v", state.Azure.Mappings)
	}
}

// TestAssignAzureState_MappingsPopulated verifies all 11 Azure mapping fields
// are populated when the server response includes them.
func TestAssignAzureState_MappingsPopulated(t *testing.T) {
	// mappings is Optional (not Computed): the prior model must carry the block
	// for it to be surfaced from the GET response.
	state := CloudIdentityProviderResourceModel{
		Azure: &cloudIdentityProviderAzureModel{Mappings: &cloudAzureMappingsModel{}},
	}
	resp := &pro.AzureConfiguration{
		CloudIDPCommon: &pro.CloudIDPCommon{ID: "az-maps"},
		Server: &pro.AzureServerConfiguration{
			TenantID: "t",
			Mappings: &pro.AzureMappings{
				UserID:     "id",
				UserName:   "userPrincipalName",
				RealName:   "displayName",
				Email:      "mail",
				Department: "department",
				Building:   "officeLocation",
				Room:       "officeName",
				Phone:      "mobilePhone",
				Position:   "jobTitle",
				GroupID:    "id",
				GroupName:  "displayName",
			},
		},
	}

	assignAzureState(&state, resp)

	maps := state.Azure.Mappings
	if maps == nil {
		t.Fatal("Mappings must not be nil")
	}
	if maps.UserID.ValueString() != "id" {
		t.Errorf("UserID mismatch")
	}
	if maps.UserName.ValueString() != "userPrincipalName" {
		t.Errorf("UserName mismatch")
	}
	if maps.Email.ValueString() != "mail" {
		t.Errorf("Email mismatch")
	}
	if maps.Phone.ValueString() != "mobilePhone" {
		t.Errorf("Phone mismatch")
	}
	if maps.GroupName.ValueString() != "displayName" {
		t.Errorf("GroupName mismatch")
	}
}

// TestAssignAzureState_NilSafe verifies that a nil response does not panic and
// leaves state untouched.
func TestAssignAzureState_NilSafe(t *testing.T) {
	state := CloudIdentityProviderResourceModel{ID: types.StringValue("kept")}
	assignAzureState(&state, nil)
	if state.ID.ValueString() != "kept" {
		t.Errorf("nil response must leave state untouched; got %q", state.ID.ValueString())
	}
}

// TestStringPtrValueOrNull_NilReturnsNull verifies stringPtrValueOrNull returns
// a null TF string when passed a nil pointer.
func TestStringPtrValueOrNull_NilReturnsNull(t *testing.T) {
	got := stringPtrValueOrNull(nil)
	if !got.IsNull() {
		t.Errorf("nil *string must map to null TF String; got %q", got.ValueString())
	}
}

// TestStringPtrValueOrNull_ValueRoundTrips verifies a non-nil pointer returns
// the correct TF string value.
func TestStringPtrValueOrNull_ValueRoundTrips(t *testing.T) {
	s := "hello"
	got := stringPtrValueOrNull(&s)
	if got.IsNull() || got.ValueString() != "hello" {
		t.Errorf("*string 'hello' must map to StringValue('hello'); got %q", got.ValueString())
	}
}
