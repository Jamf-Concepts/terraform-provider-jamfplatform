# Look up a class by ID.
data "jamfplatform_pro_class" "by_id" {
  id = "3"
}

# Look up a class by exact name.
data "jamfplatform_pro_class" "by_name" {
  name = "Biology 101"
}
