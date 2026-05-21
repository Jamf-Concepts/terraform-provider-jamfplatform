data "jamfplatform_pro_sites" "all" {}

data "jamfplatform_pro_sites" "primary_like" {
  filter = {
    name_substring = "Primary"
  }
}

output "all_sites" {
  value = data.jamfplatform_pro_sites.all.sites
}

output "primary_sites" {
  value = data.jamfplatform_pro_sites.primary_like.sites
}
