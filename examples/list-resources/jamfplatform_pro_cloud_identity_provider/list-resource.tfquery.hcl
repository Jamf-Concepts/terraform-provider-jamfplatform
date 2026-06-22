# List every Jamf Pro Cloud Identity Provider (Google and Microsoft Entra ID).
list "jamfplatform_pro_cloud_identity_provider" "all" {
  provider = jamfplatform
}

# List Cloud Identity Providers whose display name contains the substring
# "google" (case-insensitive).
list "jamfplatform_pro_cloud_identity_provider" "google" {
  provider = jamfplatform

  config {
    filter = {
      name_substring = "google"
    }
  }
}
