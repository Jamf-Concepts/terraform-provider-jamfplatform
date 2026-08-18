# Security Policy

## Reporting a vulnerability

Please report security vulnerabilities in this provider through either of these channels:

- **[Jamf's Vulnerability Disclosure Program](https://www.jamf.com/trust-center/vulnerability-disclosure/)** — Jamf's monitored intake, triaged by the Jamf Application Security team.
- **[GitHub private vulnerability reporting](https://github.com/Jamf-Concepts/terraform-provider-jamfplatform/security/advisories/new)** — private to the maintainers, and the route we use to publish an advisory once a fix ships.

Please do **not** open a public GitHub issue for a suspected vulnerability, and do not include credentials, tenant identifiers, or customer data in a report.

A useful report includes the provider version, the Terraform version, the resource or data source involved, and clear steps that demonstrate the issue.

## What to expect

We will acknowledge receipt of your report within **5 business days**. That acknowledgement confirms the report has been received and entered into triage — it is not a commitment to a remediation timeline, which depends on the severity and complexity of the issue.

We will keep you informed as triage progresses, and we will credit reporters who wish to be credited when we publish an advisory.

## Scope

**In scope:**

- The provider's own source code in this repository.
- The provider's dependencies, where the issue is reachable through this provider.
- This repository's build and release pipeline.

**Out of scope:**

- The Jamf Platform API, Jamf Pro, and other Jamf products and hosted services. Report these through [Jamf's Vulnerability Disclosure Program](https://www.jamf.com/trust-center/vulnerability-disclosure/) so they reach the right team.
- Third-party services and infrastructure Jamf does not operate.
- Social engineering, physical attacks, and denial of service.
- Findings that require an attacker to already control the machine running Terraform, or to already hold valid Jamf API credentials with the privileges the reported action needs.

## Coordinated disclosure

Please give us a reasonable opportunity to assess and address an issue before disclosing it publicly. We will work with you on timing and will publish a [GitHub Security Advisory](https://github.com/Jamf-Concepts/terraform-provider-jamfplatform/security/advisories) when a fix is released.

## Supported versions

This provider is pre-1.0 and released from `main`. Security fixes ship in a new release rather than being backported to earlier minor versions, so please upgrade to the latest release before reporting, and expect fixes to require an upgrade.

## Handling secrets when reporting

Terraform debug logging (`TF_LOG=DEBUG`) records the provider's HTTP traffic. The provider redacts credential-bearing headers and body fields before they reach the log, but please still review any log excerpt before attaching it to a report, and never share an unredacted log publicly. See the Troubleshooting section of the [README](README.md#troubleshooting) for details.
