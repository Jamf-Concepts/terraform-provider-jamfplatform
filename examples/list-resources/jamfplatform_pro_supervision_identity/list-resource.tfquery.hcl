# List every Jamf Pro supervision identity.
list "jamfplatform_pro_supervision_identity" "all" {
  provider = jamfplatform
}

# List identities whose display name contains the substring "Configurator"
# (case-insensitive).
list "jamfplatform_pro_supervision_identity" "configurator_like" {
  provider = jamfplatform

  config {
    filter = {
      name_substring = "Configurator"
    }
  }
}
