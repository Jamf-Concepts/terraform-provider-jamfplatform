# Copyright Jamf Software LLC 2026
# SPDX-License-Identifier: MPL-2.0

# Gateways are imported by gateway ID.
#
# The IPSec pre-shared key cannot be imported — Jamf Security Cloud never returns
# it — so add `ipsec.jamf_side.authentication_secret` to your configuration before the
# next apply.
terraform import jamfplatform_security_cloud_ztna_gateway.ipsec a1b2
