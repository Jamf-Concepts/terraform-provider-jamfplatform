---
page_title: "Impact alerts"
description: |-
  Show how many computers and mobile devices a change reaches, during terraform plan.
---

# Impact alerts

Jamf Pro shows an **impact alert** when you save a policy, profile, app or group, summarising how many devices the change affects. Set `impact_alerts` and the same signal appears during `terraform plan`:

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

Off by default; also settable with `JAMFPLATFORM_IMPACT_ALERTS`. Alerts are advisory — they never block a plan, and a tenant that cannot be read produces a single notice.

## What triggers an alert

Any change to an object deployed to devices, or to an object that scope is based on. Objects with no planned change stay silent.

A scope change reports what moves:

```
This change is adding 12 computers and removing 3 computers.
```

A payload change reports the audience it reaches, since adding a script to a policy affects every computer already in its scope:

```
Counted from 2 groups: All Managed Clients (44), Lab Macs (5).
The scope is unchanged; these computers will receive the updated policy.
```

Creating an object reports what will start receiving it; deleting one reports what stops.

## Reading the figure

The wording states how the number relates to reality. Jamf Pro's scope model narrows as well as broadens, so an input that cannot be evaluated does not always mean the true figure is higher.

| Wording | Meaning |
|---|---|
| `47 of 312 computers` | Every input was countable. |
| `up to 47 of 312 computers` | Something narrows the audience by an amount not determinable during a plan. |
| `47 or more of 312 computers` | Something adds devices that could not be counted. |
| `at least 47 of 312 computers` | As `47 or more` — the qualifier leads when the figure comes from a resource that can span both estates. |
| `an estimated 47 of 312 computers` | Inputs pull both ways, so the figure bounds it from neither side. |
| `47 computers`, no proportion | The scope names unmanaged devices, so a share of the managed estate would mislead. |

Anything unevaluable is named, with the direction it moves the figure:

```
Not resolvable during plan; the true figure may be lower:
  · limitations.network_segment_ids (1) — network segments are matched against a device's network location when it checks in
```

Network segments and iBeacon regions are matched against where a device is when it checks in. User-based targets reach devices through user assignment, which Jamf Pro resolves.

Scope a resource to a group this same plan creates and no figure is reported: a new group has no membership until applied, and a smart group's membership comes from its criteria rather than your configuration.

Resources targeting both estates report each side with its own denominator, because three Macs and three iPads are not the same change:

```
Impact alert — this ebook will be scoped to 3 of 4 computers and 0 of 1 mobile devices
```

## Group and class changes

Editing a group reports the knock-on effect:

```
Warning: Impact alert — this static computer group changes from 4 computers to 7 computers

Every object scoped to this static computer group applies to whatever joins it,
and stops applying to whatever leaves.
This change adds 3 computers.
```

A smart group reports its present membership and states that the outcome is decided after apply, when Jamf Pro re-evaluates the criteria against inventory.

Editing a group and something scoped to it in one plan produces two alerts, one from each side.

## Using alerts in CI

Read them from `terraform plan -json`, which carries every alert:

```bash
terraform plan -json \
  | jq -r 'select(.["@level"] == "warn")
           | .diagnostic
           | select(.summary | startswith("Impact alert"))
           | [.address, .summary] | @tsv'
```

To gate on the number, match the verb — `scoped to` for a create, `affects` for an update or delete — followed by the figure:

```
(scoped to|affects) (up to |at least |an estimated )?([0-9]+)( of [0-9]+)?
(scoped to|affects) ([0-9]+) or more( of [0-9]+)?
```

The `of` clause is optional because a figure with no proportion drops it. An alert spanning both estates joins two `X of Y` clauses with ` and ` — `3 of 4 computers and 0 of 1 mobile devices` — so match `([0-9]+) of ([0-9]+)` globally to read the second estate. Group and class alerts phrase membership instead (`will contain 7 computers`, `changes from 4 computers to 7 computers`); gate those on the arithmetic in the detail, such as `This change adds 3 computers.`

## Cost

Reads are cached for the lifetime of the plan, so two resources naming the same group read it once. A plan that changes nothing reads nothing.

| Read | When |
|---|---|
| Group list and device totals | Once per plan, on the first resource needing them |
| One group's membership | Per group named by a changing scope |
| One inventory query | Per distinct set of named devices, buildings or departments |
| Building and department names | Once per plan, if a mobile device scope names one |

These reads scale. The group list is paginated at 100 groups per request, and every request is paced by the provider's `min_request_interval_ms` (default 100ms between request starts) — a 5,000-group tenant spends around five seconds on the first alert of a plan. Membership is one paced read per distinct group named by anything changing in the plan, including groups only entering or leaving a scope, so a plan touching many groups serialises many reads.

## Scope of reporting

Every figure is a snapshot: group membership is re-evaluated continuously, so treat the number as what would happen if the change applied at that moment.

`jamfplatform_pro_vpp_assignment` and `jamfplatform_pro_vpp_invitation` raise no alert — their scope targets Jamf Pro users and user groups, with no device category to count.
