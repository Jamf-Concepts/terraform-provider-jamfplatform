data "jamfplatform_pro_sso_dependencies" "current" {}

output "sso_consumers" {
  value = [
    for dep in data.jamfplatform_pro_sso_dependencies.current.dependencies :
    "${dep.human_readable_name}: ${dep.name} (id=${dep.id})"
  ]
}
