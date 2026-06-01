# Copyright Jamf Software LLC 2026
# SPDX-License-Identifier: MPL-2.0

# Import by patch policy ID. NOTE: the optional scope and user_interaction blocks
# are not reconstructed on import (no prior state); re-declare them in config
# after importing.
terraform import jamfplatform_pro_patch_policy.example "12"
