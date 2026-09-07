# Copyright Jamf Software LLC 2026
# SPDX-License-Identifier: MPL-2.0

# User-Initiated Enrollment settings is a tenant-wide singleton. Import with
# the fixed identifier "singleton".
#
# Import leaves `mdm_signing_certificate` and `developer_certificate` unset.
# Jamf Pro reports no filename for a stored signing identity, and it keeps no
# readable copy of the WriteOnly `keystore_file` and `keystore_password` or
# their `_wo_version` rotation triggers, so nothing survives to restore them
# from.
#
# Omit a block and Terraform leaves the stored certificate in place. To remove
# a certificate, turn off the toggle that governs it,
# `signing_mdm_profile_enabled` or `sign_quickadd_package`. Declare a block
# holding a keystore and the first apply after you import uploads that keystore.
terraform import jamfplatform_pro_user_initiated_enrollment_settings.this singleton
