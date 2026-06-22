// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package sso_settings

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// These tests cover the defer-on-unknown fixes applied to the SSO settings
// config-time validators. The invariant under test (per validator): when a
// value the validator reads is UNKNOWN (variable/for_each-driven), the
// validator must DEFER (emit no diagnostic); it errors only when the value is
// genuinely null/empty, and passes for valid known input.
//
// Each defer-on-unknown regression is proven by holding every other attribute
// constant and flipping exactly ONE attribute between unknown and null: the
// unknown case must yield len(diags)==0 and the null case len(diags)>0.

// ===== tri-state helpers (shared) ===========================================

type attrState int

const (
	stNull attrState = iota
	stUnknown
	stSet
)

func strVal(s attrState, set string) tftypes.Value {
	switch s {
	case stUnknown:
		return tftypes.NewValue(tftypes.String, tftypes.UnknownValue)
	case stSet:
		return tftypes.NewValue(tftypes.String, set)
	default:
		return tftypes.NewValue(tftypes.String, nil)
	}
}

func boolVal(s attrState, set bool) tftypes.Value {
	switch s {
	case stUnknown:
		return tftypes.NewValue(tftypes.Bool, tftypes.UnknownValue)
	case stSet:
		return tftypes.NewValue(tftypes.Bool, set)
	default:
		return tftypes.NewValue(tftypes.Bool, nil)
	}
}

func stringDiagSummaries(resp validator.StringResponse) []string {
	out := make([]string, 0, len(resp.Diagnostics))
	for _, d := range resp.Diagnostics {
		out = append(out, d.Summary())
	}
	return out
}

func boolDiagSummaries(resp validator.BoolResponse) []string {
	out := make([]string, 0, len(resp.Diagnostics))
	for _, d := range resp.Diagnostics {
		out = append(out, d.Summary())
	}
	return out
}

// ===== validateSamlActiveStringNonEmpty (entity_id / group_attribute_name) ==

// samlActiveConfig builds a Config holding just configuration_type at root.
// The validated string (entity_id / group_attribute_name) arrives via
// req.ConfigValue, so it need not be part of the Config schema.
func samlActiveConfig(configType tftypes.Value) tfsdk.Config {
	objType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"configuration_type": tftypes.String,
	}}
	return tfsdk.Config{
		Schema: schema.Schema{Attributes: map[string]schema.Attribute{
			"configuration_type": schema.StringAttribute{Optional: true},
		}},
		Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
			"configuration_type": configType,
		}),
	}
}

func runSamlActiveValidator(v validator.String, attrName string, configType, value tftypes.Value) []string {
	req := validator.StringRequest{
		Path:        path.Root(attrName),
		ConfigValue: types.StringValue(""), // placeholder, overwritten below
		Config:      samlActiveConfig(configType),
	}
	// Build the validated value as a types.String reflecting the tftypes value.
	var cv types.String
	if value.IsKnown() {
		if value.IsNull() {
			cv = types.StringNull()
		} else {
			var s string
			_ = value.As(&s)
			cv = types.StringValue(s)
		}
	} else {
		cv = types.StringUnknown()
	}
	req.ConfigValue = cv

	var resp validator.StringResponse
	v.ValidateString(context.Background(), req, &resp)
	return stringDiagSummaries(resp)
}

// TestSamlEntityID_DefersWhenValueUnknown is the defer-on-unknown regression
// guard for entity_id.
func TestSamlEntityID_DefersWhenValueUnknown(t *testing.T) {
	out := runSamlActiveValidator(
		SamlEntityIDRequiredValidator(), "entity_id",
		strVal(stSet, configurationTypeSAML),
		strVal(stUnknown, ""),
	)
	if len(out) != 0 {
		t.Errorf("entity_id validator must defer when value unknown, got %v", out)
	}
}

func TestSamlEntityID_ErrorsWhenValueNull(t *testing.T) {
	out := runSamlActiveValidator(
		SamlEntityIDRequiredValidator(), "entity_id",
		strVal(stSet, configurationTypeSAML),
		strVal(stNull, ""),
	)
	if len(out) == 0 {
		t.Error("entity_id validator must error when SAML active and value null")
	}
}

func TestSamlEntityID_PassesWhenValueSet(t *testing.T) {
	out := runSamlActiveValidator(
		SamlEntityIDRequiredValidator(), "entity_id",
		strVal(stSet, configurationTypeSAML),
		strVal(stSet, "x"),
	)
	if len(out) != 0 {
		t.Errorf("entity_id validator should pass when value set, got %v", out)
	}
}

func TestSamlEntityID_SilentWhenOIDC(t *testing.T) {
	out := runSamlActiveValidator(
		SamlEntityIDRequiredValidator(), "entity_id",
		strVal(stSet, configurationTypeOIDC),
		strVal(stNull, ""),
	)
	if len(out) != 0 {
		t.Errorf("entity_id validator should not fire under OIDC, got %v", out)
	}
}

func TestSamlGroupAttributeName_DefersWhenValueUnknown(t *testing.T) {
	out := runSamlActiveValidator(
		SamlGroupAttributeNameRequiredValidator(), "group_attribute_name",
		strVal(stSet, configurationTypeSAML),
		strVal(stUnknown, ""),
	)
	if len(out) != 0 {
		t.Errorf("group_attribute_name validator must defer when value unknown, got %v", out)
	}
}

func TestSamlGroupAttributeName_ErrorsWhenValueNull(t *testing.T) {
	out := runSamlActiveValidator(
		SamlGroupAttributeNameRequiredValidator(), "group_attribute_name",
		strVal(stSet, configurationTypeSAML),
		strVal(stNull, ""),
	)
	if len(out) == 0 {
		t.Error("group_attribute_name validator must error when SAML active and value null")
	}
}

func TestSamlGroupAttributeName_PassesWhenValueSet(t *testing.T) {
	out := runSamlActiveValidator(
		SamlGroupAttributeNameRequiredValidator(), "group_attribute_name",
		strVal(stSet, configurationTypeSAML),
		strVal(stSet, "x"),
	)
	if len(out) != 0 {
		t.Errorf("group_attribute_name validator should pass when value set, got %v", out)
	}
}

func TestSamlGroupAttributeName_SilentWhenOIDC(t *testing.T) {
	out := runSamlActiveValidator(
		SamlGroupAttributeNameRequiredValidator(), "group_attribute_name",
		strVal(stSet, configurationTypeOIDC),
		strVal(stNull, ""),
	)
	if len(out) != 0 {
		t.Errorf("group_attribute_name validator should not fire under OIDC, got %v", out)
	}
}

// ===== configurationTypeBlockValidator ======================================
//
// Reads path.Root("saml_settings") and path.Root("oidc_settings") as objects;
// only their presence matters. Minimal but schema-consistent object shapes.

var cfgSamlObjType = tftypes.Object{AttributeTypes: map[string]tftypes.Type{
	"entity_id": tftypes.String,
}}

var cfgOidcObjType = tftypes.Object{AttributeTypes: map[string]tftypes.Type{
	"user_mapping": tftypes.String,
}}

var cfgRootObjType = tftypes.Object{AttributeTypes: map[string]tftypes.Type{
	"configuration_type": tftypes.String,
	"saml_settings":      cfgSamlObjType,
	"oidc_settings":      cfgOidcObjType,
}}

func configTypeConfig(configType, samlSettings, oidcSettings tftypes.Value) tfsdk.Config {
	samlSchema := schema.SingleNestedAttribute{
		Optional: true,
		Attributes: map[string]schema.Attribute{
			"entity_id": schema.StringAttribute{Optional: true},
		},
	}
	oidcSchema := schema.SingleNestedAttribute{
		Optional: true,
		Attributes: map[string]schema.Attribute{
			"user_mapping": schema.StringAttribute{Optional: true},
		},
	}
	return tfsdk.Config{
		Schema: schema.Schema{Attributes: map[string]schema.Attribute{
			"configuration_type": schema.StringAttribute{Optional: true},
			"saml_settings":      samlSchema,
			"oidc_settings":      oidcSchema,
		}},
		Raw: tftypes.NewValue(cfgRootObjType, map[string]tftypes.Value{
			"configuration_type": configType,
			"saml_settings":      samlSettings,
			"oidc_settings":      oidcSettings,
		}),
	}
}

func samlObj(present bool) tftypes.Value {
	if !present {
		return tftypes.NewValue(cfgSamlObjType, nil)
	}
	return tftypes.NewValue(cfgSamlObjType, map[string]tftypes.Value{
		"entity_id": tftypes.NewValue(tftypes.String, "ent"),
	})
}

func oidcObj(present bool) tftypes.Value {
	if !present {
		return tftypes.NewValue(cfgOidcObjType, nil)
	}
	return tftypes.NewValue(cfgOidcObjType, map[string]tftypes.Value{
		"user_mapping": tftypes.NewValue(tftypes.String, "EMAIL"),
	})
}

func runConfigTypeValidator(configType, samlSettings, oidcSettings tftypes.Value) []string {
	req := validator.StringRequest{
		Path:        path.Root("configuration_type"),
		ConfigValue: tftypesToTypesString(configType),
		Config:      configTypeConfig(configType, samlSettings, oidcSettings),
	}
	var resp validator.StringResponse
	ConfigurationTypeBlockValidator().ValidateString(context.Background(), req, &resp)
	return stringDiagSummaries(resp)
}

func tftypesToTypesString(v tftypes.Value) types.String {
	if !v.IsKnown() {
		return types.StringUnknown()
	}
	if v.IsNull() {
		return types.StringNull()
	}
	var s string
	_ = v.As(&s)
	return types.StringValue(s)
}

// TestConfigTypeBlock_DefersWhenSamlUnknown is the SAML-branch defer-on-unknown
// regression guard: configuration_type = SAML with saml_settings UNKNOWN must
// NOT error (the block is not proven absent).
func TestConfigTypeBlock_DefersWhenSamlUnknown(t *testing.T) {
	out := runConfigTypeValidator(
		strVal(stSet, configurationTypeSAML),
		tftypes.NewValue(cfgSamlObjType, tftypes.UnknownValue),
		oidcObj(false),
	)
	if len(out) != 0 {
		t.Errorf("configuration_type validator must defer when saml_settings unknown, got %v", out)
	}
}

func TestConfigTypeBlock_ErrorsWhenSamlNull(t *testing.T) {
	out := runConfigTypeValidator(
		strVal(stSet, configurationTypeSAML),
		samlObj(false),
		oidcObj(false),
	)
	if len(out) == 0 {
		t.Error("configuration_type = SAML must error when saml_settings null")
	}
}

func TestConfigTypeBlock_OIDCForbidsSamlPresent(t *testing.T) {
	out := runConfigTypeValidator(
		strVal(stSet, configurationTypeOIDC),
		samlObj(true),
		oidcObj(true),
	)
	if len(out) == 0 {
		t.Error("configuration_type = OIDC must error when saml_settings present")
	}
}

func TestConfigTypeBlock_OIDCErrorsWhenOidcNull(t *testing.T) {
	out := runConfigTypeValidator(
		strVal(stSet, configurationTypeOIDC),
		samlObj(false),
		oidcObj(false),
	)
	if len(out) == 0 {
		t.Error("configuration_type = OIDC must error when oidc_settings null")
	}
}

// TestConfigTypeBlock_DefersWhenOidcUnknown: OIDC with oidc_settings UNKNOWN
// and saml_settings null must NOT error. saml_settings is held null so the
// forbidden-check stays quiet, isolating the oidc_settings defer.
func TestConfigTypeBlock_DefersWhenOidcUnknown(t *testing.T) {
	out := runConfigTypeValidator(
		strVal(stSet, configurationTypeOIDC),
		samlObj(false),
		tftypes.NewValue(cfgOidcObjType, tftypes.UnknownValue),
	)
	if len(out) != 0 {
		t.Errorf("configuration_type validator must defer when oidc_settings unknown, got %v", out)
	}
}

// ===== metadataSourceBranchValidator ========================================
//
// Reads idp_url, federation_metadata_file, metadata_file_name as siblings of
// metadata_source under a parent object (saml_settings).

var mdParentObjType = tftypes.Object{AttributeTypes: map[string]tftypes.Type{
	"metadata_source":          tftypes.String,
	"idp_url":                  tftypes.String,
	"federation_metadata_file": tftypes.String,
	"metadata_file_name":       tftypes.String,
}}

var mdRootObjType = tftypes.Object{AttributeTypes: map[string]tftypes.Type{
	"saml_settings": mdParentObjType,
}}

func metadataConfig(source, idpURL, fmf, mfn tftypes.Value) tfsdk.Config {
	parentSchema := schema.SingleNestedAttribute{
		Optional: true,
		Attributes: map[string]schema.Attribute{
			"metadata_source":          schema.StringAttribute{Optional: true},
			"idp_url":                  schema.StringAttribute{Optional: true},
			"federation_metadata_file": schema.StringAttribute{Optional: true},
			"metadata_file_name":       schema.StringAttribute{Optional: true},
		},
	}
	return tfsdk.Config{
		Schema: schema.Schema{Attributes: map[string]schema.Attribute{
			"saml_settings": parentSchema,
		}},
		Raw: tftypes.NewValue(mdRootObjType, map[string]tftypes.Value{
			"saml_settings": tftypes.NewValue(mdParentObjType, map[string]tftypes.Value{
				"metadata_source":          source,
				"idp_url":                  idpURL,
				"federation_metadata_file": fmf,
				"metadata_file_name":       mfn,
			}),
		}),
	}
}

func runMetadataValidator(source, idpURL, fmf, mfn tftypes.Value) []string {
	req := validator.StringRequest{
		Path:        path.Root("saml_settings").AtName("metadata_source"),
		ConfigValue: tftypesToTypesString(source),
		Config:      metadataConfig(source, idpURL, fmf, mfn),
	}
	var resp validator.StringResponse
	MetadataSourceBranchValidator().ValidateString(context.Background(), req, &resp)
	return stringDiagSummaries(resp)
}

// TestMetadata_URL_DefersWhenIdpURLUnknown: URL branch, idp_url UNKNOWN (others
// null) must NOT error.
func TestMetadata_URL_DefersWhenIdpURLUnknown(t *testing.T) {
	out := runMetadataValidator(
		strVal(stSet, metadataSourceURL),
		strVal(stUnknown, ""),
		strVal(stNull, ""),
		strVal(stNull, ""),
	)
	if len(out) != 0 {
		t.Errorf("metadata_source = URL must defer when idp_url unknown, got %v", out)
	}
}

func TestMetadata_URL_ErrorsWhenIdpURLNull(t *testing.T) {
	out := runMetadataValidator(
		strVal(stSet, metadataSourceURL),
		strVal(stNull, ""),
		strVal(stNull, ""),
		strVal(stNull, ""),
	)
	if len(out) == 0 {
		t.Error("metadata_source = URL must error when idp_url null")
	}
}

// TestMetadata_FILE_DefersWhenFmfUnknown: FILE requires BOTH
// federation_metadata_file AND metadata_file_name. To isolate the fmf defer,
// metadata_file_name is held set and idp_url null.
func TestMetadata_FILE_DefersWhenFmfUnknown(t *testing.T) {
	out := runMetadataValidator(
		strVal(stSet, metadataSourceFILE),
		strVal(stNull, ""),
		strVal(stUnknown, ""),
		strVal(stSet, "idp.xml"),
	)
	if len(out) != 0 {
		t.Errorf("metadata_source = FILE must defer when federation_metadata_file unknown, got %v", out)
	}
}

func TestMetadata_FILE_ErrorsWhenFmfNull(t *testing.T) {
	out := runMetadataValidator(
		strVal(stSet, metadataSourceFILE),
		strVal(stNull, ""),
		strVal(stNull, ""),
		strVal(stSet, "idp.xml"),
	)
	if len(out) == 0 {
		t.Error("metadata_source = FILE must error when federation_metadata_file null")
	}
}

// ===== idpProviderTypeOtherValidator ========================================

var idpParentObjType = tftypes.Object{AttributeTypes: map[string]tftypes.Type{
	"idp_provider_type":        tftypes.String,
	"other_provider_type_name": tftypes.String,
}}

var idpRootObjType = tftypes.Object{AttributeTypes: map[string]tftypes.Type{
	"saml_settings": idpParentObjType,
}}

func idpConfig(provider, other tftypes.Value) tfsdk.Config {
	parentSchema := schema.SingleNestedAttribute{
		Optional: true,
		Attributes: map[string]schema.Attribute{
			"idp_provider_type":        schema.StringAttribute{Optional: true},
			"other_provider_type_name": schema.StringAttribute{Optional: true},
		},
	}
	return tfsdk.Config{
		Schema: schema.Schema{Attributes: map[string]schema.Attribute{
			"saml_settings": parentSchema,
		}},
		Raw: tftypes.NewValue(idpRootObjType, map[string]tftypes.Value{
			"saml_settings": tftypes.NewValue(idpParentObjType, map[string]tftypes.Value{
				"idp_provider_type":        provider,
				"other_provider_type_name": other,
			}),
		}),
	}
}

func runIdpValidator(provider, other tftypes.Value) []string {
	req := validator.StringRequest{
		Path:        path.Root("saml_settings").AtName("idp_provider_type"),
		ConfigValue: tftypesToTypesString(provider),
		Config:      idpConfig(provider, other),
	}
	var resp validator.StringResponse
	IdpProviderTypeOtherValidator().ValidateString(context.Background(), req, &resp)
	return stringDiagSummaries(resp)
}

func TestIdpOther_DefersWhenNameUnknown(t *testing.T) {
	out := runIdpValidator(strVal(stSet, "OTHER"), strVal(stUnknown, ""))
	if len(out) != 0 {
		t.Errorf("idp_provider_type = OTHER must defer when other_provider_type_name unknown, got %v", out)
	}
}

func TestIdpOther_ErrorsWhenNameNull(t *testing.T) {
	out := runIdpValidator(strVal(stSet, "OTHER"), strVal(stNull, ""))
	if len(out) == 0 {
		t.Error("idp_provider_type = OTHER must error when other_provider_type_name null")
	}
}

func TestIdpOther_PassesWhenNameSet(t *testing.T) {
	out := runIdpValidator(strVal(stSet, "OTHER"), strVal(stSet, "x"))
	if len(out) != 0 {
		t.Errorf("idp_provider_type = OTHER should pass when name set, got %v", out)
	}
}

// ===== userAttributeEnabledValidator (Bool) =================================

var uaParentObjType = tftypes.Object{AttributeTypes: map[string]tftypes.Type{
	"user_attribute_enabled": tftypes.Bool,
	"user_attribute_name":    tftypes.String,
}}

var uaRootObjType = tftypes.Object{AttributeTypes: map[string]tftypes.Type{
	"saml_settings": uaParentObjType,
}}

func userAttrConfig(enabled, name tftypes.Value) tfsdk.Config {
	parentSchema := schema.SingleNestedAttribute{
		Optional: true,
		Attributes: map[string]schema.Attribute{
			"user_attribute_enabled": schema.BoolAttribute{Optional: true},
			"user_attribute_name":    schema.StringAttribute{Optional: true},
		},
	}
	return tfsdk.Config{
		Schema: schema.Schema{Attributes: map[string]schema.Attribute{
			"saml_settings": parentSchema,
		}},
		Raw: tftypes.NewValue(uaRootObjType, map[string]tftypes.Value{
			"saml_settings": tftypes.NewValue(uaParentObjType, map[string]tftypes.Value{
				"user_attribute_enabled": enabled,
				"user_attribute_name":    name,
			}),
		}),
	}
}

func tftypesToTypesBool(v tftypes.Value) types.Bool {
	if !v.IsKnown() {
		return types.BoolUnknown()
	}
	if v.IsNull() {
		return types.BoolNull()
	}
	var b bool
	_ = v.As(&b)
	return types.BoolValue(b)
}

func runUserAttrValidator(enabled, name tftypes.Value) []string {
	req := validator.BoolRequest{
		Path:        path.Root("saml_settings").AtName("user_attribute_enabled"),
		ConfigValue: tftypesToTypesBool(enabled),
		Config:      userAttrConfig(enabled, name),
	}
	var resp validator.BoolResponse
	UserAttributeEnabledValidator().ValidateBool(context.Background(), req, &resp)
	return boolDiagSummaries(resp)
}

func TestUserAttr_DefersWhenNameUnknown(t *testing.T) {
	out := runUserAttrValidator(boolVal(stSet, true), strVal(stUnknown, ""))
	if len(out) != 0 {
		t.Errorf("user_attribute_enabled = true must defer when name unknown, got %v", out)
	}
}

func TestUserAttr_ErrorsWhenNameNull(t *testing.T) {
	out := runUserAttrValidator(boolVal(stSet, true), strVal(stNull, ""))
	if len(out) == 0 {
		t.Error("user_attribute_enabled = true must error when name null")
	}
}

func TestUserAttr_PassesWhenNameSet(t *testing.T) {
	out := runUserAttrValidator(boolVal(stSet, true), strVal(stSet, "x"))
	if len(out) != 0 {
		t.Errorf("user_attribute_enabled = true should pass when name set, got %v", out)
	}
}

// ===== groupEnrollmentAccessEnabledValidator (Bool) =========================
//
// Reads path.Root("sso_for_enrollment_enabled") and
// path.Root("group_enrollment_access_name").

var geRootObjType = tftypes.Object{AttributeTypes: map[string]tftypes.Type{
	"group_enrollment_access_enabled": tftypes.Bool,
	"sso_for_enrollment_enabled":      tftypes.Bool,
	"group_enrollment_access_name":    tftypes.String,
}}

func groupEnrollConfig(enabled, ssoForEnroll, name tftypes.Value) tfsdk.Config {
	return tfsdk.Config{
		Schema: schema.Schema{Attributes: map[string]schema.Attribute{
			"group_enrollment_access_enabled": schema.BoolAttribute{Optional: true},
			"sso_for_enrollment_enabled":      schema.BoolAttribute{Optional: true},
			"group_enrollment_access_name":    schema.StringAttribute{Optional: true},
		}},
		Raw: tftypes.NewValue(geRootObjType, map[string]tftypes.Value{
			"group_enrollment_access_enabled": enabled,
			"sso_for_enrollment_enabled":      ssoForEnroll,
			"group_enrollment_access_name":    name,
		}),
	}
}

func runGroupEnrollValidator(enabled, ssoForEnroll, name tftypes.Value) []string {
	req := validator.BoolRequest{
		Path:        path.Root("group_enrollment_access_enabled"),
		ConfigValue: tftypesToTypesBool(enabled),
		Config:      groupEnrollConfig(enabled, ssoForEnroll, name),
	}
	var resp validator.BoolResponse
	GroupEnrollmentAccessEnabledValidator().ValidateBool(context.Background(), req, &resp)
	return boolDiagSummaries(resp)
}

func TestGroupEnroll_DefersWhenNameUnknown(t *testing.T) {
	out := runGroupEnrollValidator(
		boolVal(stSet, true),
		boolVal(stSet, true),
		strVal(stUnknown, ""),
	)
	if len(out) != 0 {
		t.Errorf("group_enrollment_access_enabled must defer when name unknown, got %v", out)
	}
}

func TestGroupEnroll_ErrorsWhenNameNull(t *testing.T) {
	out := runGroupEnrollValidator(
		boolVal(stSet, true),
		boolVal(stSet, true),
		strVal(stNull, ""),
	)
	if len(out) == 0 {
		t.Error("group_enrollment_access_enabled must error when name null and sso_for_enrollment_enabled true")
	}
}

// ===== signingCertificateSetupTypeValidator =================================
//
// Reads type, key, keystore_file, keystore_file_name, keystore_password,
// password as siblings of setup_type under a parent object (signing_certificate).

var scParentObjType = tftypes.Object{AttributeTypes: map[string]tftypes.Type{
	"setup_type":         tftypes.String,
	"type":               tftypes.String,
	"key":                tftypes.String,
	"keystore_file":      tftypes.String,
	"keystore_file_name": tftypes.String,
	"keystore_password":  tftypes.String,
	"password":           tftypes.String,
}}

var scRootObjType = tftypes.Object{AttributeTypes: map[string]tftypes.Type{
	"signing_certificate": scParentObjType,
}}

func signingConfig(setupType, typ, key, ksFile, ksFileName, ksPass, pass tftypes.Value) tfsdk.Config {
	parentSchema := schema.SingleNestedAttribute{
		Optional: true,
		Attributes: map[string]schema.Attribute{
			"setup_type":         schema.StringAttribute{Optional: true},
			"type":               schema.StringAttribute{Optional: true},
			"key":                schema.StringAttribute{Optional: true},
			"keystore_file":      schema.StringAttribute{Optional: true},
			"keystore_file_name": schema.StringAttribute{Optional: true},
			"keystore_password":  schema.StringAttribute{Optional: true},
			"password":           schema.StringAttribute{Optional: true},
		},
	}
	return tfsdk.Config{
		Schema: schema.Schema{Attributes: map[string]schema.Attribute{
			"signing_certificate": parentSchema,
		}},
		Raw: tftypes.NewValue(scRootObjType, map[string]tftypes.Value{
			"signing_certificate": tftypes.NewValue(scParentObjType, map[string]tftypes.Value{
				"setup_type":         setupType,
				"type":               typ,
				"key":                key,
				"keystore_file":      ksFile,
				"keystore_file_name": ksFileName,
				"keystore_password":  ksPass,
				"password":           pass,
			}),
		}),
	}
}

func runSigningValidator(setupType, typ, key, ksFile, ksFileName, ksPass, pass tftypes.Value) []string {
	req := validator.StringRequest{
		Path:        path.Root("signing_certificate").AtName("setup_type"),
		ConfigValue: tftypesToTypesString(setupType),
		Config:      signingConfig(setupType, typ, key, ksFile, ksFileName, ksPass, pass),
	}
	var resp validator.StringResponse
	SigningCertificateSetupTypeValidator().ValidateString(context.Background(), req, &resp)
	return stringDiagSummaries(resp)
}

// TestSigning_DefersWhenOneSiblingUnknown: setup_type = UPLOADED with `type`
// UNKNOWN and all other five required siblings set must NOT error.
func TestSigning_DefersWhenOneSiblingUnknown(t *testing.T) {
	out := runSigningValidator(
		strVal(stSet, setupTypeUploaded),
		strVal(stUnknown, ""), // type
		strVal(stSet, "k"),    // key
		strVal(stSet, "f"),    // keystore_file
		strVal(stSet, "n"),    // keystore_file_name
		strVal(stSet, "p"),    // keystore_password
		strVal(stSet, "pw"),   // password
	)
	if len(out) != 0 {
		t.Errorf("setup_type = UPLOADED must defer when a required sibling is unknown, got %v", out)
	}
}

func TestSigning_ErrorsWhenAllSiblingsNull(t *testing.T) {
	out := runSigningValidator(
		strVal(stSet, setupTypeUploaded),
		strVal(stNull, ""),
		strVal(stNull, ""),
		strVal(stNull, ""),
		strVal(stNull, ""),
		strVal(stNull, ""),
		strVal(stNull, ""),
	)
	if len(out) == 0 {
		t.Error("setup_type = UPLOADED must error when required siblings null")
	}
}

// ===== requiresSamlBoolValidator ============================================
//
// Reads path.Root("configuration_type"). The validated flag arrives via
// req.ConfigValue. The rule: a feature flag set true is rejected in pure OIDC
// mode (the server silently coerces it to false) and allowed when SAML is part
// of the configuration. Null/unknown flag, or unknown configuration_type, defer.

func runRequiresSamlValidator(flag, configType tftypes.Value) []string {
	objType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"configuration_type": tftypes.String,
	}}
	req := validator.BoolRequest{
		Path:        path.Root("sso_for_enrollment_enabled"),
		ConfigValue: tftypesToTypesBool(flag),
		Config: tfsdk.Config{
			Schema: schema.Schema{Attributes: map[string]schema.Attribute{
				"configuration_type": schema.StringAttribute{Optional: true},
			}},
			Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
				"configuration_type": configType,
			}),
		},
	}
	var resp validator.BoolResponse
	RequiresSamlBoolValidator("sso_for_enrollment_enabled").ValidateBool(context.Background(), req, &resp)
	return boolDiagSummaries(resp)
}

func TestRequiresSaml_ErrorsWhenTrueAndOIDC(t *testing.T) {
	out := runRequiresSamlValidator(boolVal(stSet, true), strVal(stSet, configurationTypeOIDC))
	if len(out) == 0 {
		t.Error("flag = true must error when configuration_type = OIDC")
	}
}

func TestRequiresSaml_SilentWhenFalseAndOIDC(t *testing.T) {
	out := runRequiresSamlValidator(boolVal(stSet, false), strVal(stSet, configurationTypeOIDC))
	if len(out) != 0 {
		t.Errorf("flag = false must pass in OIDC, got %v", out)
	}
}

func TestRequiresSaml_PassesWhenTrueAndSAML(t *testing.T) {
	out := runRequiresSamlValidator(boolVal(stSet, true), strVal(stSet, configurationTypeSAML))
	if len(out) != 0 {
		t.Errorf("flag = true must pass when configuration_type = SAML, got %v", out)
	}
}

func TestRequiresSaml_PassesWhenTrueAndOIDCWithSAML(t *testing.T) {
	out := runRequiresSamlValidator(boolVal(stSet, true), strVal(stSet, configurationTypeOIDCWithSAML))
	if len(out) != 0 {
		t.Errorf("flag = true must pass when configuration_type = OIDC_WITH_SAML, got %v", out)
	}
}

func TestRequiresSaml_DefersWhenConfigTypeUnknown(t *testing.T) {
	out := runRequiresSamlValidator(boolVal(stSet, true), strVal(stUnknown, ""))
	if len(out) != 0 {
		t.Errorf("must defer when configuration_type unknown, got %v", out)
	}
}

func TestRequiresSaml_SilentWhenFlagNull(t *testing.T) {
	out := runRequiresSamlValidator(boolVal(stNull, false), strVal(stSet, configurationTypeOIDC))
	if len(out) != 0 {
		t.Errorf("must defer when flag null, got %v", out)
	}
}
