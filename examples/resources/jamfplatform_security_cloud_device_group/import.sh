# Copyright Jamf Software LLC 2026
# SPDX-License-Identifier: MPL-2.0

# Device groups are imported by group ID. The built-in "Default Group" cannot be
# imported — Jamf Security Cloud gives it no identifier.
terraform import jamfplatform_security_cloud_device_group.executives 57497e81-d499-4f99-8fe8-8f262d0f5b8f
