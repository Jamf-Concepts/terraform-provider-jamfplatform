# Copyright Jamf Software LLC 2026
# SPDX-License-Identifier: MPL-2.0

# Import a Cloud Identity Provider by its Jamf Pro ID. The provider type
# (Google or Azure) is discovered automatically from the registry on import.
# Note: the Google keystore (file, password, wo_version) cannot be recovered
# on import — re-supply them in configuration.
terraform import jamfplatform_pro_cloud_identity_provider.example "1"
