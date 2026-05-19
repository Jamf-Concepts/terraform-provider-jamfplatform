data "jamfplatform_pro_scripts" "before_run" {
  filter = [
    {
      selector = "priority"
      argument = "BEFORE"
    }
  ]
}

data "jamfplatform_pro_scripts" "by_name_prefix" {
  filter = [
    {
      selector = "name"
      argument = "Cleanup*"
    }
  ]
}

output "before_run_scripts" {
  value = data.jamfplatform_pro_scripts.before_run.scripts
}
