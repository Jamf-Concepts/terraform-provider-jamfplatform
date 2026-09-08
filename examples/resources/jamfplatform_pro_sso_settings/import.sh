# Copyright Jamf Software LLC 2026
# SPDX-License-Identifier: MPL-2.0

# SSO settings is a tenant-wide singleton. Import with the fixed identifier
# "singleton".
#
# Import leaves `signing_certificate` unset even when your tenant holds a
# certificate, and does so on purpose. Declaring that block is how you hand the
# certificate to Terraform to manage; if import handed it over for you, later
# dropping the block would delete your tenant's certificate. So omit the block
# and Terraform leaves the stored certificate alone.
#
# Declare `setup_type = "GENERATED"` against a tenant that already holds a
# generated certificate and nothing happens. Jamf Pro mints no replacement and
# the serial number holds steady. Declare `setup_type = "UPLOADED"` and
# Terraform re-sends your keystore, because Jamf Pro reports no readable copy of
# the `_wo_version` rotation triggers and the values you configure always differ
# from the empty ones in state.
#
# For the rest of what an import records, and why the first plan afterwards can
# propose removing blocks you never wrote, see the "Importing existing objects"
# guide.
terraform import jamfplatform_pro_sso_settings.this singleton
