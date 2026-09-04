# Look up a supervision identity by ID.
data "jamfplatform_pro_supervision_identity" "by_id" {
  id = "1"
}

# Or look up by display name. Display names are not required to be unique. This
# errors if more than one identity shares the name; use id to disambiguate.
data "jamfplatform_pro_supervision_identity" "by_name" {
  display_name = "Apple Configurator Identity"
}

output "supervision_identity_common_name" {
  value = data.jamfplatform_pro_supervision_identity.by_name.common_name
}

output "supervision_identity_expiration" {
  value = data.jamfplatform_pro_supervision_identity.by_name.expiration_date
}
