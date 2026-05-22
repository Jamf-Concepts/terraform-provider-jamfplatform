data "jamfplatform_pro_user_groups" "all" {}

data "jamfplatform_pro_user_groups" "vpp_related" {
  filter = {
    name_substring = "VPP"
  }
}

output "all_user_groups" {
  value = data.jamfplatform_pro_user_groups.all.user_groups
}

output "vpp_user_groups" {
  value = data.jamfplatform_pro_user_groups.vpp_related.user_groups
}
