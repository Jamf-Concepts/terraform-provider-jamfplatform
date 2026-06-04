# Look up an ebook by exact name.
data "jamfplatform_pro_ebook" "by_name" {
  name = "Field Operations Guide"
}

# Or look up by ID.
data "jamfplatform_pro_ebook" "by_id" {
  id = "79"
}

output "ebook_url" {
  value = data.jamfplatform_pro_ebook.by_name.url
}
