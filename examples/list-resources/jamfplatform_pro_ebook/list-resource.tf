# List all ebooks (identity-only: id + display name). Requires Terraform query
# support. The optional name_substring filter is applied client-side.
list "jamfplatform_pro_ebook" "all" {
  provider = jamfplatform

  config {
    filter = {
      name_substring = "guide"
    }
  }
}
