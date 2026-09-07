# Copyright Jamf Software LLC 2026
# SPDX-License-Identifier: MPL-2.0

# Import an existing AD CS integration by its Jamf Pro AD CS Settings ID.
#
# The certificate blocks are restored as far as Jamf Pro allows: `filename` is
# read back, and the `*_details` blocks carry the stored certificate's subject,
# issuer, serial number and expiry. The WriteOnly inputs (`data_wo`,
# `password_wo`) and the `wo_version` rotation trigger have no server-side
# equivalent, so they come back unset and must be re-declared.
#
# Expect the first plan after importing to show `+ wo_version` and the first
# apply to re-send the certificates. That is unavoidable: an INBOUND
# integration must declare both blocks, `data_wo` requires `wo_version`
# alongside it, and Jamf Pro never returns the version that was last sent — so
# the configured version always differs from the empty one in state. Re-sending
# the same certificate is harmless; it replaces the stored copy with an
# identical one and the serial number does not change.
terraform import jamfplatform_pro_pki_adcs.inbound "25"
