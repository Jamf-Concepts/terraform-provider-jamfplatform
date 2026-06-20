// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package sso_settings

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ssoSettingsTimeoutAttributeTypes defines the timeout attribute types.
var ssoSettingsTimeoutAttributeTypes = map[string]attr.Type{
	"create": types.StringType,
	"read":   types.StringType,
	"update": types.StringType,
	"delete": types.StringType,
}

// configurationType values accepted by /v3/sso. UNKNOWN appears in the
// upstream openapi enum but is rejected at runtime (probe G7); the schema
// restricts to the three accepted values.
const (
	configurationTypeSAML         = "SAML"
	configurationTypeOIDC         = "OIDC"
	configurationTypeOIDCWithSAML = "OIDC_WITH_SAML"
)

// validConfigurationTypes are the accepted configuration_type values.
var validConfigurationTypes = []string{
	configurationTypeSAML,
	configurationTypeOIDC,
	configurationTypeOIDCWithSAML,
}

// metadataSource values accepted by /v3/sso. UNKNOWN appears in openapi but
// is runtime-rejected (probe G7).
const (
	metadataSourceURL  = "URL"
	metadataSourceFILE = "FILE"
)

var validMetadataSources = []string{metadataSourceURL, metadataSourceFILE}

// idpProviderType enum values.
var validIdpProviderTypes = []string{
	"ADFS", "OKTA", "GOOGLE", "SHIBBOLETH", "ONELOGIN",
	"PING", "CENTRIFY", "AZURE", "OTHER",
}

// userMapping enum (shared between SAML and OIDC blocks).
var validUserMappings = []string{"USERNAME", "EMAIL"}

// signing_certificate.setup_type enum.
const (
	setupTypeGenerated = "GENERATED"
	setupTypeUploaded  = "UPLOADED"
	setupTypeNone      = "NONE"
)

var validSetupTypes = []string{setupTypeGenerated, setupTypeUploaded, setupTypeNone}

// signing_certificate.type enum.
var validKeystoreTypes = []string{"PKCS12", "JKS"}

// signingCertificateKeyAttrTypes describes the element type of the Computed
// `keys` list in the cert sub-block. The wire shape is
// `[{id: string, valid: bool}]` per pro.CertificateKey.
var signingCertificateKeyAttrTypes = map[string]attr.Type{
	"id":    types.StringType,
	"valid": types.BoolType,
}
