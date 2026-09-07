# Copyright Jamf Software LLC 2026
# SPDX-License-Identifier: MPL-2.0

# User-Initiated Enrollment settings is a tenant-wide singleton. Import with
# the fixed identifier "singleton".
#
# The `mdm_signing_certificate` and `developer_certificate` blocks are not
# restored by import. Jamf Pro returns no filename for a stored signing
# identity, and the WriteOnly `keystore_file` / `keystore_password` and their
# `_wo_version` rotation triggers have no server-side equivalent, so there is
# nothing to restore them from.
#
# A configuration that omits a block leaves the stored certificate in place —
# the toggle (`signing_mdm_profile_enabled`, `sign_quickadd_package`) is what
# removes it, not the block. Declaring a block with a keystore re-sends that
# keystore on the first apply after importing.
terraform import jamfplatform_pro_user_initiated_enrollment_settings.this singleton
