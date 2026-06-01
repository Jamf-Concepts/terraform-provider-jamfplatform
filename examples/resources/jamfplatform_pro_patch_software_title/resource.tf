# A patch software title is defined by its catalog key (name_id) and patch
# source (source_id). The server populates the full list of available_versions;
# assign packages to specific versions via version_packages so patch policies
# can target them.

resource "jamfplatform_pro_patch_software_title" "eight_by_eight" {
  name      = "8x8 Work"
  name_id   = "285"
  source_id = 1

  category_id        = "-1"
  site_id            = "-1"
  web_notification   = true
  email_notification = false

  # Map software_version -> package ID. Only the versions you list are managed;
  # removing a key clears that version's package on the next apply.
  version_packages = {
    "8.33.2.2" = jamfplatform_pro_package.work_8_33.id
  }
}

resource "jamfplatform_pro_package" "work_8_33" {
  display_name = "8x8 Work 8.33.2.2"
  file_name    = "8x8-work-8.33.2.2.pkg"
}
