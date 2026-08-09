---
page_title: "Impact alerts"
description: |-
  Show how many computers and mobile devices a change reaches, during terraform plan.
---

# Impact alerts

Jamf Pro shows an **impact alert** when you save a policy, a configuration profile, an app or a group, summarising how many devices the change affects. Set `impact_alerts` on the provider and the same signal appears during `terraform plan`:

```hcl
provider "jamfplatform" {
  impact_alerts = true
}
```

```
Warning: Impact alert — this policy will be scoped to 47 of 312 computers (15%)

  with jamfplatform_pro_policy.security_baseline,
  on main.tf line 12, in resource "jamfplatform_pro_policy" "security_baseline":
  12:   scope = {

Counted from 2 groups: All Managed Clients (44), Lab Macs (5).
Less 1 group excluded: Contractors (2).

Group membership is a snapshot taken during this plan and can change before
or during apply.
```

Off by default. It can also be enabled with the `JAMFPLATFORM_IMPACT_ALERTS` environment variable.

Alerts are **advisory only**. They never block a plan, and a tenant that cannot be read produces a single notice rather than an error.

## What triggers an alert

Any change to an object that is deployed to devices, or to an object that scope is based on. That includes changes which leave the audience alone:

```
Warning: Impact alert — this policy affects 47 of 312 computers (15%)

Counted from 2 groups: All Managed Clients (44), Lab Macs (5).
The scope is unchanged; these computers will receive the updated policy.
```

Adding a script to a policy reaches every computer in its scope, so it is reported even though nobody entered or left. Objects with no planned change stay silent.

A scope change reports what moves, which is the figure Jamf Pro's own alert leads with:

```
This change is adding 12 computers and removing 3 computers.
```

Deleting an object reports what stops receiving it. Creating one reports what will start.

## Reading the figure

The wording states how the number relates to reality. Jamf Pro's scope model narrows as well as broadens, so an input the provider cannot evaluate does not always mean the true figure is higher.

| Wording | Meaning |
|---|---|
| `47 of 312 computers` | Every input was countable. |
| `up to 47 of 312 computers` | Something narrows the audience by an amount that cannot be determined during a plan. |
| `47 or more of 312 computers` | Something adds devices that could not be counted. |
| `an estimated 47 of 312 computers` | Inputs pull in both directions, so the figure bounds it from neither side. |
| `47 computers` (no proportion) | The scope names devices that are not managed, so a proportion of the managed estate would be meaningless. |

Whatever cannot be evaluated is named, with the direction it moves the figure in:

```
Not resolvable during plan; the true figure may be lower:
  · limitations.network_segment_ids (1) — network segments are matched against a device's network location when it checks in
Not resolvable during plan; the true figure may be higher:
  · targets.user_group_ids (1) — user-based targets reach devices through user assignment, which Jamf Pro resolves
```

Network segments and iBeacon regions are matched against where a device is when it checks in, so they have no membership ahead of time. User-based targets reach devices through user assignment, which Jamf Pro resolves.

An object scoped to a group that this same plan creates reports no figure at all — a new group has no membership until it has been applied, and a smart group's membership is decided by Jamf Pro from its criteria rather than by your configuration.

Resources that target both estates report each side with its own denominator, because three Macs and three iPads are not the same change:

```
Impact alert — this ebook will be scoped to 3 of 4 computers and 0 of 1 mobile devices
```

## Group and class changes

Editing a group or a class reports the knock-on effect rather than the group itself:

```
Warning: Impact alert — this static computer group changes from 4 computers to 7 computers

Every object scoped to this static computer group applies to whatever joins it,
and stops applying to whatever leaves.
This change adds 3 computers.
```

A smart group reports its present membership and says plainly that the outcome is not knowable, because Jamf Pro re-evaluates the criteria against inventory after the change is applied.

## Every figure is a snapshot

Group membership is re-evaluated continuously, so the number affected can change between plan and apply. A plan that edits a group **and** something scoped to that group produces two alerts, one from each side — a resource cannot see a sibling's changes, so the scoped object's figure is computed from the group's membership as it stands now.

Treat the number as what would happen if the change applied at that moment, not as a promise.

## Using alerts in CI

Diagnostics appear in `terraform plan -json`. They are **not** in `terraform show -json`, which carries resource changes rather than diagnostics — a gate built on `show -json` will never fire.

```bash
terraform plan -json \
  | jq -r 'select(.["@level"] == "warn")
           | .diagnostic
           | select(.summary | startswith("Impact alert"))
           | [.address, .summary] | @tsv'
```

Two things to know before building a gate on this:

Terraform consolidates warnings that share identical summary text in its human output, showing one plus *"and N more similar warnings elsewhere"*. The `-json` stream always carries all of them, so use that.

The figure exists only as prose in `summary`, so extracting it means matching all four phrasings:

```
"scoped to (up to |an estimated |)?([0-9]+) of ([0-9]+)"
"scoped to ([0-9]+) or more of ([0-9]+)"
```

## Cost

Enabling impact alerts adds reads to each plan. The provider paces outbound requests, so the cost is roughly proportional to the number of reads rather than to their latency:

| Read | When |
|---|---|
| Group list and device totals | Once per plan, on the first resource that needs them |
| One group's membership | Once per group named by a changing scope |
| One inventory query | Once per distinct set of named devices, buildings or departments |
| Building and department names | Once per plan, only if a mobile device scope names one |

Everything is cached for the lifetime of the plan, so two resources naming the same group read it once. A plan that changes nothing performs no reads at all.

## What is not reported

`jamfplatform_pro_vpp_assignment` and `jamfplatform_pro_vpp_invitation` raise no alert. Their scope targets Jamf Pro users and user groups, with no device or device-group category, so the alert could only ever say that the figure cannot be determined.
