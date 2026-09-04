data "jamfplatform_ai_governance_policies" "all" {
  sort = ["name:asc"]
}

# The policies worth reviewing after a tool publishes a new settings schema.
data "jamfplatform_ai_governance_policies" "needs_review" {
  schema_drift_only = true
}

output "policies_with_unpublished_changes" {
  description = "Policies holding a draft that has not been published."
  value = [
    for policy in data.jamfplatform_ai_governance_policies.all.policies :
    policy.name if policy.has_draft
  ]
}

output "policies_on_an_older_schema" {
  description = "Policies written against a settings schema version that is no longer current."
  value       = [for policy in data.jamfplatform_ai_governance_policies.needs_review.policies : policy.name]
}
