# Copyright Jamf Software LLC 2026
# SPDX-License-Identifier: MPL-2.0

# SSO settings is a tenant-wide singleton. Import with the fixed identifier
# "singleton".
#
# The `signing_certificate` block is deliberately NOT restored by import, even
# when the tenant holds a certificate. That block is how Terraform is told to
# manage the certificate, and adopting one it was never asked to manage would
# make dropping the block from the configuration delete the tenant's
# certificate. Import therefore leaves it unset, and a configuration that omits
# it leaves the stored certificate alone.
#
# Declaring `setup_type = "GENERATED"` after importing a tenant that already has
# a generated certificate is a no-op — Jamf Pro is not asked to mint a new one,
# so the serial number is stable. Declaring `setup_type = "UPLOADED"` re-sends
# the keystore, because the `_wo_version` rotation triggers have no server-side
# equivalent and the configured values always differ from the empty ones in
# state.
terraform import jamfplatform_pro_sso_settings.this singleton
