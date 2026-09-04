# Pure OIDC, SSO disabled: minimal scaffolding.
resource "jamfplatform_pro_sso_settings" "oidc_disabled" {
  sso_enabled                                          = false
  sso_bypass_allowed                                   = false
  sso_for_enrollment_enabled                           = false
  sso_for_macos_self_service_enabled                   = false
  enrollment_sso_for_account_driven_enrollment_enabled = false
  group_enrollment_access_enabled                      = false

  configuration_type = "OIDC"

  oidc_settings = {
    user_mapping                   = "EMAIL"
    jamf_id_authentication_enabled = true
  }
}

# SAML + IdP metadata via URL (Jamf Pro fetches the document at apply time).
resource "jamfplatform_pro_sso_settings" "saml_url" {
  sso_enabled                                          = true
  sso_bypass_allowed                                   = true
  sso_for_enrollment_enabled                           = true
  sso_for_macos_self_service_enabled                   = true
  enrollment_sso_for_account_driven_enrollment_enabled = false
  group_enrollment_access_enabled                      = false

  configuration_type = "SAML"

  saml_settings = {
    idp_provider_type    = "OKTA"
    entity_id            = "/saml/metadata"
    metadata_source      = "URL"
    idp_url              = "https://trial-9344750.okta.com/app/exks2co0x41zYbk8y697/sso/saml/metadata"
    session_timeout      = 480
    user_mapping         = "EMAIL"
    group_attribute_name = "http://schemas.xmlsoap.org/claims/Group"
  }
}

# SAML + IdP metadata via file. The IdP-issued metadata XML is read from
# disk and base64-encoded at plan time.
resource "jamfplatform_pro_sso_settings" "saml_file" {
  sso_enabled                                          = true
  sso_bypass_allowed                                   = true
  sso_for_enrollment_enabled                           = false
  sso_for_macos_self_service_enabled                   = false
  enrollment_sso_for_account_driven_enrollment_enabled = false
  group_enrollment_access_enabled                      = false

  configuration_type = "SAML"

  saml_settings = {
    idp_provider_type        = "ADFS"
    entity_id                = "/saml/metadata"
    metadata_source          = "FILE"
    federation_metadata_file = filebase64("${path.module}/idp-metadata.xml")
    metadata_file_name       = "idp-metadata.xml"
    user_mapping             = "EMAIL"
    group_attribute_name     = "http://schemas.xmlsoap.org/claims/Group"
  }
}

# SAML + uploaded signing keystore. `keystore_password` and `password` are
# WriteOnly. Bump `_wo_version` to rotate.
resource "jamfplatform_pro_sso_settings" "saml_uploaded_cert" {
  sso_enabled                                          = true
  sso_bypass_allowed                                   = true
  sso_for_enrollment_enabled                           = false
  sso_for_macos_self_service_enabled                   = false
  enrollment_sso_for_account_driven_enrollment_enabled = false
  group_enrollment_access_enabled                      = false

  configuration_type = "SAML"

  saml_settings = {
    idp_provider_type    = "OKTA"
    entity_id            = "/saml/metadata"
    metadata_source      = "URL"
    idp_url              = "https://trial-9344750.okta.com/app/exks2co0x41zYbk8y697/sso/saml/metadata"
    user_mapping         = "EMAIL"
    group_attribute_name = "http://schemas.xmlsoap.org/claims/Group"
  }

  signing_certificate = {
    setup_type         = "UPLOADED"
    type               = "PKCS12"
    key                = "jamf-saml"
    keystore_file      = filebase64("${path.module}/jamf-saml.p12")
    keystore_file_name = "jamf-saml.p12"

    keystore_password            = var.sso_keystore_password
    keystore_password_wo_version = 1

    password            = var.sso_key_password
    password_wo_version = 1
  }
}

variable "sso_keystore_password" {
  type      = string
  sensitive = true
}

variable "sso_key_password" {
  type      = string
  sensitive = true
}
