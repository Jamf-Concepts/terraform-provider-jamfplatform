# Copyright Jamf Software LLC 2026
# SPDX-License-Identifier: MPL-2.0

# A tenant holds one UEM Connect integration, so where one already exists, import it
# rather than declaring a second — a second create is refused.

# The integration's ID is not a value anyone would have written down, so let
# Terraform find it and generate the import block:
#
#   terraform query -generate-config-out=uem_connect.tf
#
# with a query block naming the list resource:
#
#   list "jamfplatform_security_cloud_uem_connect" "existing" {
#     provider = jamfplatform
#   }

# Or import by ID directly, which the data source reports:
#
#   data "jamfplatform_security_cloud_uem_connect" "existing" {}
#   output "id" { value = data.jamfplatform_security_cloud_uem_connect.existing.id }
terraform import jamfplatform_security_cloud_uem_connect.jamf_pro 6a91b958619ef153a5a63d72
