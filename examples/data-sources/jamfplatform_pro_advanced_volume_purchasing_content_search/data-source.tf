# Look up an advanced volume purchasing content search by ID.
data "jamfplatform_pro_advanced_volume_purchasing_content_search" "by_id" {
  id = "483"
}

# Look up an advanced volume purchasing content search by exact name.
data "jamfplatform_pro_advanced_volume_purchasing_content_search" "by_name" {
  name = "Office apps with available licenses"
}
