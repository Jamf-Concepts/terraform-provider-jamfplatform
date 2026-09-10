---
page_title: "Moving to the jamf namespace"
description: |-
  This provider now publishes as jamf/jamfplatform. What to run in each workspace to move a configuration off jamf-concepts/jamfplatform.
---

# Moving to the jamf namespace

This provider now publishes to the Terraform Registry as `jamf/jamfplatform`. Earlier releases
carried `jamf-concepts/jamfplatform`.

**Editing `source` alone will not move a configuration across.** Terraform records the registry
namespace in state, alongside every resource the provider manages. To Terraform,
`jamf/jamfplatform` and `jamf-concepts/jamfplatform` are two separate providers, so the state has
to be rewritten before a configuration can name the new one.

Staying put works too. A configuration pinned to `jamf-concepts/jamfplatform` goes on planning and
applying against the releases published there. It stops gaining new ones.

## Moving a configuration across

Three steps, in this order, for **each workspace and each state file** holding `jamfplatform`
resources. Rewrite the state first:

```console
$ terraform state replace-provider jamf-concepts/jamfplatform jamf/jamfplatform
```

It lists what it will rewrite, asks for confirmation, and writes a state backup before touching
anything. Add `-auto-approve` to run it unattended in a pipeline.

Then point `source` at the new namespace, leaving any version constraint as it is:

```hcl
terraform {
  required_providers {
    jamfplatform = {
      source = "jamf/jamfplatform"
    }
  }
}
```

Then run `terraform init`.

OpenTofu works the same way, with `tofu` in place of `terraform`. The provider publishes to the
OpenTofu Registry under the same `jamf` namespace.

Coming from `v0.28.1` or earlier? [Upgrading to the Platform API GA](platform-api-ga) covers
everything else that configuration needs: the gateway host, the credentials, the scope attribute,
and the constructs the GA removed.

## Things that are easy to miss

**One run per state file.** A configuration applied across several workspaces has a state file per
workspace, and `replace-provider` rewrites whichever one is selected. Select each in turn, or run
the command once per backend key.

**Modules pin `source` too.** A module with its own `required_providers` block naming
`jamf-concepts/jamfplatform` needs the same edit. Terraform reports a provider requirement it
cannot satisfy, which says nothing about the namespace.

**Rewrite the state before you edit `source`.** `replace-provider` reads and writes state, with no
network access and no provider binary involved, so it runs happily while `source` still names the
old namespace. Edit `source` first and `terraform init` has nowhere to go.

**Nothing about your Jamf tenant changes.** Only the registry address moves. The same credentials,
`base_url`, scope attribute and resources apply on either side of it.

## Reference

- [`terraform state replace-provider`](https://developer.hashicorp.com/terraform/cli/commands/state/replace-provider)
