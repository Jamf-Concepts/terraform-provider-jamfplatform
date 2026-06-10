# Returns every Jamf Protect plan previously synced into Jamf Pro, with each
# plan's associated configuration profile. The catalog reflects the most
# recent plans sync (it persists even after unregistering, so rows may be
# stale on an unregistered tenant).
data "jamfplatform_pro_jamf_protect_plans" "all" {}

# Optional server-side RSQL filter and sort.
data "jamfplatform_pro_jamf_protect_plans" "by_name" {
  filter = [
    {
      selector = "name"
      argument = "Default Plan"
    }
  ]
  sort = ["name:asc"]
}

output "protect_plan_profiles" {
  value = {
    for plan in data.jamfplatform_pro_jamf_protect_plans.all.plans :
    plan.name => plan.profile_name
  }
}
