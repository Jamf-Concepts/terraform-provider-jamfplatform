# Look up a policy by exact name.
data "jamfplatform_pro_policy" "by_name" {
  name = "tf-acc-minimal-policy"
}

output "policy_id" {
  value = data.jamfplatform_pro_policy.by_name.id
}

# Look up a policy by ID.
data "jamfplatform_pro_policy" "by_id" {
  id = "42"
}
