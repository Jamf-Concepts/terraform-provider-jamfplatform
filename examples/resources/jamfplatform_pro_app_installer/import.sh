# Copyright Jamf Software LLC 2026
# SPDX-License-Identifier: MPL-2.0

# Import an existing App Installer deployment by its Jamf Pro ID.
#
# WARNING: read the first plan after this import before you apply it.
#
# Import records the Self Service and notification settings Jamf Pro reports,
# even where your configuration declares none, so the plan proposes removing
# them. On this resource that removal is real. Terraform writes both blocks
# whole, so Jamf Pro resets whatever you leave out of your configuration, and
# the description, deadline and notification messages you never declared are
# gone after the apply.
#
# Declare what you want to keep before applying. Copy the values out of the plan
# output into your configuration and plan again. See the "Importing existing
# objects" guide.
terraform import jamfplatform_pro_app_installer.example "177"
