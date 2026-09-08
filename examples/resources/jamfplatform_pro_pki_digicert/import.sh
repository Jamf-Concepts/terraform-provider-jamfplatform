# Copyright Jamf Software LLC 2026
# SPDX-License-Identifier: MPL-2.0

# Import an existing DigiCert integration by its Jamf Pro DigiCert Settings ID.
#
# Import restores `client_certificate.filename`, plus the subject, issuer,
# serial number and expiry under `client_certificate_details`. It cannot
# restore the WriteOnly `data_wo` and `password_wo`, or the `wo_version`
# rotation trigger, because Jamf Pro stores no readable copy of any of them.
#
# Omit `wo_version` and the first apply after you import sends nothing, adopting
# the certificate Jamf Pro already holds. Set it and Terraform re-sends the
# certificate, which is what bumping it asks for.
#
# For the rest of what an import records, and why the first plan afterwards can
# propose removing blocks you never wrote, see the "Importing existing objects"
# guide.
terraform import jamfplatform_pro_pki_digicert.example "24"
