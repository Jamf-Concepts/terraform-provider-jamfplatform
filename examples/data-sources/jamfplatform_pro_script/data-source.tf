data "jamfplatform_pro_script" "example_by_id" {
  id = "42"
}

output "script_example_by_id" {
  value     = data.jamfplatform_pro_script.example_by_id
  sensitive = true
}
