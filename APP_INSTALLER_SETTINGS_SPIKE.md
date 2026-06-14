# App Installer Global Settings — Spike Doc
# Probed: 2026-06-14

## Final Schema

```hcl
resource "jamfplatform_pro_app_installer_settings" "example" {
  deployment_settings {
    batch_size       = 1000        # Optional Int64; UI enum 100/500/1000/5000/10000, server accepts any
    batch_frequency  = 60          # Optional Int64 (minutes); server enforces 10–1440
    days             = ["MONDAY", "WEDNESDAY", "FRIDAY"]  # Optional Set(String)
    server_time_from = "08:00:00Z" # Optional String; must include Z suffix
    server_time_to   = "17:00:00Z" # Optional String; must include Z suffix
  }
  end_user_experience {
    notification_frequency  = 2      # Optional Int64; unit = HOURS
    notification_message    = "An update is pending."
    update_deadline         = 24     # Optional Int64; unit = HOURS
    force_quit_message      = "Please quit and save your work."
    force_quit_grace_period = 10     # Optional Int64; unit = MINUTES (wire-confirmed 2026-06-14)
    update_complete_message = "Update complete."
    relaunch = true   # Optional Bool
    suppress = false  # Optional Bool
  }
}
```

## Key Wire Findings

### Full-replace null=clear semantics (confirmed)
- PUT `{}` → both blocks cleared to all-null
- PUT `{"deploymentProcessControls":{}}` → all fields in that block cleared
- Omitting a field within a managed block = reset that field to null (field-level full-replace)

### Block-level omit = preserve (GET-merge)
Omitting an entire top-level block does NOT clear the server's values. The resource
does GET → merge plan over current → PUT (singleton adoption pattern, §805 STYLE_GUIDE).
Block-level: omit = preserve. Field-level within a managed block: omit = clear.

### Block presence semantics (value-based normalization)
The server ALWAYS returns both blocks in GET responses, even at factory defaults
(both blocks present with all-null fields). The TF state assigner normalizes
"block with all-null fields" → nil TF block.

Implementation: **value-based normalization** — a block is nil iff ALL leaf fields
are null in the server response. State-gating rejected: breaks import (prior state nil).

### Units (wire-confirmed)
- `notificationInterval` → `notification_frequency`: **HOURS**
- `deadline` → `update_deadline`: **HOURS**
- `quitDelay` → `force_quit_grace_period`: **MINUTES** (wire-probe: UI set 10 min → wire returns 10)

### commandsBatchSize → batch_size
Server accepts freeform integer. UI enum (100/500/1000/5000/10000) is display-only.
Decision: plain Optional Int64Attribute with AtLeast(1).

### fromTimeOfDay / toTimeOfDay → server_time_from / server_time_to
Only `"HH:MM:SSZ"` format accepted.
Regex validator: `^([01][0-9]|2[0-3]):[0-5][0-9]:[0-5][0-9]Z$`

### daysOfWeek → days
- null (omitted) → server stores/returns null
- [] → server stores/returns [] (distinct from null)
- Enum: MONDAY, TUESDAY, WEDNESDAY, THURSDAY, FRIDAY, SATURDAY, SUNDAY

## Schema Design Decisions

### 1. UI-aligned field names (per STYLE_GUIDE §Jamf Pro Resource Naming)
All field names follow Jamf Pro UI labels, not SDK/wire names:
- Tab 1 "Deployment settings" → block `deployment_settings`
- Tab 2 "End user experience" → block `end_user_experience`
- "Batch size" → `batch_size`
- "Batch frequency" → `batch_frequency` (unit documented in description)
- "Server time" From/To → `server_time_from` / `server_time_to`
- "Days" → `days`
- "Notification frequency" → `notification_frequency`
- "Update deadline" → `update_deadline`
- "Force quit message" → `force_quit_message`
- "Force quit grace period" → `force_quit_grace_period`
- "Update complete message" → `update_complete_message`

### 2. Both blocks: Optional-only (§815 pattern)
All leaf attrs are nullable with no server defaults → Optional-only (no Optional+Computed).
Block omit = preserve (via GET-merge). Leaf omit within block = clear (field-level full-replace).

### 3. Validators
- `batch_frequency`: `int64validator.Between(10, 1440)`
- `batch_size`: `int64validator.AtLeast(1)`
- `notification_frequency`, `update_deadline`, `force_quit_grace_period`: `int64validator.AtLeast(1)`
- `days` elements: `stringvalidator.OneOf(7 day names)` via `setvalidator.ValueStringsAre`
- `server_time_from`, `server_time_to`: regex `^([01][0-9]|2[0-3]):[0-5][0-9]:[0-5][0-9]Z$`

### 4. types.Set for days
Not types.List — order not meaningful for day-of-week membership.

### 5. No unofficial API note in schema
The SDK handles this internally; provider schema descriptions do not mention it.

## Files
```
internal/resources/pro/settings/app_installer_settings/
  crud.go
  data_source.go
  input_builders.go
  input_builders_test.go
  model_types.go
  resource.go
  resource_acceptance_test.go
  schema_test.go
  schema_types.go
  state_builders.go
  state_builders_test.go
```
