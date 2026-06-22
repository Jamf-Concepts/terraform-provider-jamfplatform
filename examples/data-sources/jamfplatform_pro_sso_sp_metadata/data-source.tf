data "jamfplatform_pro_sso_sp_metadata" "current" {}

# Service Provider metadata is consumed by the IdP administrator when
# registering Jamf Pro as a relying party. Empty in pure OIDC mode.
output "sp_metadata_xml" {
  value = data.jamfplatform_pro_sso_sp_metadata.current.xml
}
