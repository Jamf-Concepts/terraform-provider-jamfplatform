# Copyright Jamf Software LLC 2026
# SPDX-License-Identifier: MPL-2.0

# Import a supervision identity by its Jamf Pro ID. The write-only password and
# certificate are not part of state, so they are not populated by import.
terraform import jamfplatform_pro_supervision_identity.generated "1"
