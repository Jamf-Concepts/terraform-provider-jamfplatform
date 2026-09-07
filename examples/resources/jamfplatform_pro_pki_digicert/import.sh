# Copyright Jamf Software LLC 2026
# SPDX-License-Identifier: MPL-2.0

# Import an existing DigiCert integration by its Jamf Pro DigiCert Settings ID.
#
# The `client_certificate` block is restored as far as Jamf Pro allows:
# `filename` is read back, and `client_certificate_details` carries the stored
# certificate's subject, issuer, serial number and expiry. The WriteOnly inputs
# (`data_wo`, `password_wo`) and the `wo_version` rotation trigger have no
# server-side equivalent, so they come back unset.
#
# Leave `wo_version` out of the configuration and the first apply after
# importing sends nothing — the imported certificate is adopted as-is. Set it,
# and the certificate is re-sent, which is what bumping it means.
terraform import jamfplatform_pro_pki_digicert.example "24"
