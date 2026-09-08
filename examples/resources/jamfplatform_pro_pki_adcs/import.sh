# Copyright Jamf Software LLC 2026
# SPDX-License-Identifier: MPL-2.0

# Import an existing AD CS integration by its Jamf Pro AD CS Settings ID.
#
# Import restores what Jamf Pro returns: `filename` on each certificate block,
# plus the subject, issuer, serial number and expiry under `*_details`. It
# cannot restore the WriteOnly `data_wo` and `password_wo`, or the `wo_version`
# rotation trigger, because Jamf Pro stores no readable copy of any of them.
# Re-declare those three in your configuration.
#
# The first plan after you import will show `+ wo_version`, and the first apply
# will re-send both certificates. You cannot avoid that. An INBOUND integration
# has to declare both blocks, `data_wo` obliges you to set `wo_version` next to
# it, and Jamf Pro never reports which version it last received, so the version
# you configure always differs from the empty one in state. Re-sending costs
# you nothing: Jamf Pro replaces the stored certificate with an identical copy
# and the serial number stays the same.
#
# For the rest of what an import records, and why the first plan afterwards can
# propose removing blocks you never wrote, see the "Importing existing objects"
# guide.
terraform import jamfplatform_pro_pki_adcs.inbound "25"
