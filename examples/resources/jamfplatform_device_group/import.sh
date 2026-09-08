# Copyright Jamf Software LLC 2026
# SPDX-License-Identifier: MPL-2.0

# Import an existing device group by its ID.
#
# Import records every optional attribute the platform reports, including a
# static group's members and its description, so the first plan afterwards can
# propose removing what your configuration does not declare. See the "Importing
# existing objects" guide for what applying that plan does.
terraform import jamfplatform_device_group.example "3ff60bb2-95dd-48f1-9141-0de74f5ad18c"
