data "jamfplatform_pro_vpp_invitation" "by_id" {
  id = "2"
}

data "jamfplatform_pro_vpp_invitation" "by_name" {
  name = "Volume Purchasing Self Service"
}

output "vpp_invitation_distribution_method" {
  value = data.jamfplatform_pro_vpp_invitation.by_name.distribution_method
}
