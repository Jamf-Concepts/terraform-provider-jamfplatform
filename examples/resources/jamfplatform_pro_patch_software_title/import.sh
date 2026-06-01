# Copyright Jamf Software LLC 2026
# SPDX-License-Identifier: MPL-2.0

# Import by patch software title ID. NOTE: version_packages is a managed-subset
# map that cannot be reconstructed on import (no prior state); re-declare it in
# config after importing.
terraform import jamfplatform_pro_patch_software_title.example "6"
