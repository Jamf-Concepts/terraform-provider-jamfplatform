# Query for Jamf Pro scripts that run BEFORE policy actions
list "jamfplatform_pro_script" "before_run_scripts" {
  provider = jamfplatform

  config {
    filter {
      selector = "priority"
      argument = "BEFORE"
    }
  }
}

# Query for scripts whose name starts with a substring
list "jamfplatform_pro_script" "scripts_by_name_prefix" {
  provider = jamfplatform

  config {
    filter {
      selector = "name"
      argument = "Cleanup*"
    }
  }
}
