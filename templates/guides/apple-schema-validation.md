---
page_title: "Apple schema validation"
description: |-
  Blueprint payloads and declarations are checked against Apple's schemas during terraform plan, and every finding is an error.
---

# Apple schema validation

The provider carries Apple's own configuration profile and declarative device management schemas, generated from [apple/device-management](https://github.com/apple/device-management), and checks blueprint payloads against them during `terraform plan`.

**This is a breaking change.** A configuration that planned cleanly on `v0.32.0` and earlier can now fail its plan. Nothing about what the provider sends changed, and no state is rewritten — only the plan outcome.

Two things moved:

- **Legacy configuration profile payload findings were warnings. They are now errors.** `component_blocks[].legacy_payloads` was checked before, but a finding that the embedded schemas could explain printed as a warning and the apply went ahead.
- **Declaration payloads are checked for the first time.** Both `component_blocks[].apple_declarations` and `component_blocks[].custom_declarations` now have their `payload` validated against the declaration type Apple publishes.

If your plan is clean, you have nothing to do.

## Why a finding is an error

Nothing else reports it. Wire probing found that the blueprints service accepts a declaration carrying an invented key, a wrong-cased key, a value of the wrong type or a value outside Apple's declared range, answers `201`, and a deploy of the same blueprint answers `202` and reports SUCCEEDED. The Jamf Pro editor then renders the declaration with the offending setting blank, and the device never receives it. A legacy payload behaves the same way for an unrecognised key.

So the failure is silent at every layer you can see: the API said yes, the deploy said yes, and the setting is not on the device. A warning would leave you with a blueprint that reports success and does nothing, which is why the plan stops instead.

Declaration key names are matched **case-sensitively**, and a wrong-cased key is discarded. Legacy configuration profile payloads behave the opposite way: Jamf Pro silently restores Apple's spelling, which is why a miscased key there shows up as a plan that never converges rather than as a setting that never applies. Do not carry a spelling that worked in a profile into a declaration on the strength of that.

## The findings

| Finding | What it means | Can an old snapshot explain it? |
|---|---|---|
| Unknown Apple declaration type / unrecognised legacy payload type | The name is absent from the embedded schemas | Yes |
| Unknown key in payload | Apple declares no such key for this type | Yes |
| Unknown status item | A status subscription names an item Apple does not publish | Yes |
| Value not in enum | The value is outside the closed set Apple declares | Yes — Apple widens sets between revisions |
| Value out of range | A number falls outside Apple's declared bounds | Yes — Apple widens bounds between revisions |
| Missing required key | A key Apple marks required is absent | Yes — Apple relaxes `required` between revisions |
| Wrong value type | A string where a boolean is declared, and so on | No |
| Miscased type or key name | The name matches a declared one apart from case | No |
| Declaration kind does not match its type | `kind` disagrees with the declaration type's prefix | No |

The right-hand column is the one to read first. A finding marked **Yes** says so in its own detail, names the upstream branches and commits the schemas came from, and points at the escape hatch. A finding marked **No** is wrong against every revision of Apple's schema that declares the name at all, so the provider's age cannot be the cause.

The schemas are the union of Apple's `release` branch and its newest `seed_OS_*` (pre-release) branch, because Jamf's generative-declarations service tracks seed: keys and declaration types appear in the Jamf Pro editor while still pre-release. A finding about a pre-release part of the schema says so.

## A key the Jamf Pro editor offers, rejected by the plan

Upgrade the provider. The schemas are **embedded in the provider binary and frozen at the release you have installed**, so there is no cache to clear and no attribute to set. Upstream is refreshed daily and released with the provider, so the window between Apple publishing a key and the provider carrying it is short, but it is not zero.

Until an upgrade is available, deliver the payload or declaration through `raw_component`, which is checked against nothing.

## Delivering a declaration unchecked

Move the declaration into `raw_component` under the identifier the typed component would have used, and JSON-encode the configuration:

```hcl
component_blocks = [
  {
    name = "App Settings"
    raw_component = [
      {
        identifier = "com.jamf.ddm-strict"
        configuration = {
          declarations = jsonencode([
            {
              channelType = "SYSTEM"
              kind        = "CONFIGURATION"
              type        = "com.apple.configuration.app.settings"
              payloadKey  = 1
              payload = {
                Allowed = { DeniedApps = ["com.apple.screenshots"] }
              }
            },
          ])
        }
      },
    ]
  },
]
```

Set `payloadKey` yourself. It is the 1-based position of the declaration within the request, and it is what a `$PAYLOAD_<n>` cross-reference from another declaration resolves against. The typed components derive it from list order; `raw_component` derives nothing, so a declaration without a key cannot be referenced.

## Delivering a legacy configuration profile payload unchecked

**Move the whole block's payloads, not the one that failed.** Every `legacy_payloads` entry in a component block folds into a single `com.jamf.ddm-configuration-profile` component whose `payloadContent` is the array of payloads. Split them and the apply writes that component twice, and the payloads you left in `legacy_payloads` stop being reconciled against the server, so drift on those goes unreported.

Before:

```hcl
component_blocks = [
  {
    name = "Safari Restrictions"
    legacy_payloads = [
      {
        payload_type = "com.apple.applicationaccess"
        settings = jsonencode({
          allowSafariHistoryClearing = false
          allowSafariPrivateBrowsing = false
        })
      },
      {
        payload_type = "com.apple.dock"
        settings     = jsonencode({ tilesize = 48 })
      },
    ]
  },
]
```

After:

```hcl
component_blocks = [
  {
    name = "Safari Restrictions"
    raw_component = [
      {
        identifier = "com.jamf.ddm-configuration-profile"
        configuration = {
          payloadDisplayName = "Safari Restrictions"
          payloadContent = jsonencode([
            {
              payloadType       = "com.apple.applicationaccess"
              payloadIdentifier = "1f9c07a4-3b7e-4c21-9f0d-7a5c8e2b6d41"

              allowSafariHistoryClearing = false
              allowSafariPrivateBrowsing = false
            },
            {
              payloadType       = "com.apple.dock"
              payloadIdentifier = "6b2d51e8-90af-4d3c-8c17-2ea94f60b7d3"

              tilesize = 48
            },
          ])
        }
      },
    ]
  },
]
```

Three things to carry across correctly:

- **Each payload's settings sit alongside `payloadType`**, not nested under a `settings` key. The typed attribute merges them in.
- **`payloadIdentifier` is per payload and required.** The typed attribute derives one from the payload type; `raw_component` does not, so state one and do not change it between applies — Jamf Pro keys the stored payload on it.
- **`payloadDisplayName` is per component.** The typed attribute uses the blueprint's own name.

Moving a block to `raw_component` shows up in the plan as one component destroyed and another created. Read it before you apply.

## Further reading

- [Apple's declarative device management schemas](https://github.com/apple/device-management/tree/release/declarative) — the source of every declaration finding.
- [Apple's configuration profile schemas](https://github.com/apple/device-management/tree/release/mdm/profiles) — the source of every legacy payload finding.
- [Blueprints Guide](https://learn.jamf.com/r/en-US/Jamf-Blueprints-Guide) — the capability itself.
