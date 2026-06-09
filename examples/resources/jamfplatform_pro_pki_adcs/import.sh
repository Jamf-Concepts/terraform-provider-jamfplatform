# Copyright Jamf Software LLC 2026
# SPDX-License-Identifier: MPL-2.0

# Import an existing AD CS integration by its Jamf Pro AD CS Settings ID.
# WriteOnly certificate fields (data_wo / password_wo) and wo_version are not
# populated by import; re-declare them in config to manage rotation.
terraform import jamfplatform_pro_pki_adcs.inbound "25"
