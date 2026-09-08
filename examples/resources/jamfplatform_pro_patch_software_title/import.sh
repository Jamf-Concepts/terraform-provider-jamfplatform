# Copyright Jamf Software LLC 2026
# SPDX-License-Identifier: MPL-2.0

# Import by patch software title ID.
#
# Import records no version_packages: Terraform manages only the versions you
# declare, and every package already on the title stays assigned. Declare the
# ones you want it to own and leave the rest to the admin UI.
terraform import jamfplatform_pro_patch_software_title.example "6"
