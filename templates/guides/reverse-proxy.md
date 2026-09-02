---
page_title: "Reverse proxies and custom headers"
description: |-
  Reach Jamf through a proxy that authenticates callers itself, using custom_headers and authorization_header_name.
---

# Reverse proxies and custom headers

Some proxies authenticate callers themselves. Two attributes cover that case:

- `custom_headers` — extra headers on every request, including the token request.
- `authorization_header_name` — send the Jamf credential in a header other than `Authorization`.

Leave both unset to talk to Jamf directly.

## First: do you need them?

Probably not. An ordinary forward proxy needs neither. Set the standard variables and stop:

```sh
export HTTPS_PROXY=http://proxy.internal:3128
export NO_PROXY=localhost,127.0.0.1
```

## Adding headers

```hcl
provider "jamfplatform" {
  custom_headers = {
    "X-Proxy-Route" = "eu-west"
  }
}
```

Or from the environment, one `Name: value` pair per line:

```sh
export JAMFPLATFORM_CUSTOM_HEADERS="X-Proxy-Route: eu-west
X-Trace-Id: $CI_JOB_ID"
```

Set the attribute and the variable goes unread.

Notes:

- Names are case-insensitive. Supplying both `x-proxy-route` and `X-Proxy-Route` is an error, not a silent drop.
- Only the first colon on a line splits name from value, so values may contain colons. A line with no colon is an error.
- The whole map is sensitive. Terraform cannot redact one entry.

## Two credentials on one request

The proxy wants `Authorization` for itself and expects the Jamf credential elsewhere:

```hcl
provider "jamfplatform" {
  custom_headers = {
    "Authorization" = "Basic ${var.proxy_basic_credential}"
  }
  authorization_header_name = "X-Jamf-Authorization"
}
```

| Header | Value | Read by |
|---|---|---|
| `Authorization` | `Basic …` | the proxy |
| `X-Jamf-Authorization` | `Bearer …` | Jamf |

Only the destination changes. The provider still obtains and validates the credential as usual, so a wrong `client_secret` fails where it always did.

Confirm with `TF_LOG=INFO terraform plan`. It logs header names, never values:

```
Jamf Platform provider configured: credential_header=X-Jamf-Authorization
custom_headers=["Authorization", "X-Proxy-Route"]
```

## Refused at configure time

All of these are refused up front. Let through, each fails later with an error naming the wrong cause.

| Setting | Refused because |
|---|---|
| `X-Environment-Id` / `X-Tenant-Id` in `custom_headers` | Set from `environment_id` / `tenant_id`. An overridden scope gets the same error as a wrong credential. |
| `Cookie` in `custom_headers` | Would displace Jamf Cloud's session cookie. See below. |
| `Content-Type` / `Accept` in `custom_headers` | Would replace the media type each call picks for itself. See below. |
| `authorization_header_name = "Authorization"` | Moving a header onto itself removes it; every call then answers 401, as a wrong client secret does. |
| `authorization_header_name` set to a scope header | The credential overwrites the scope; Jamf answers `403 OWNERSHIP_FORBIDDEN`. |
| `authorization_header_name` matching a `custom_headers` key | The custom value replaces the credential. |
| `authorization_header_name` set to `Content-Type` or `Accept` | The credential replaces the media type, leaving a request Jamf cannot parse. |

`User-Agent` is accepted, then dropped with a warning: the provider sets its own afterwards.

### Why `Cookie` is refused

Jamf Cloud uses a [sticky session cookie](https://developer.jamf.com/jamf-pro/docs/sticky-sessions-for-jamf-cloud) to keep a client on one node, so a read after a write sees the write. Supplied headers replace; they do not add. A `Cookie` entry throws that pinning away, and the symptom is not an error about cookies. It is an occasional `Provider produced inconsistent result after apply`, against whichever resource lost the race.

A proxy needing its own cookie needs nothing here. The provider stores cookies the proxy sets and returns them on later requests. No other custom header affects session pinning.

### Why `Content-Type` and `Accept` are refused

Same structural reason as `Cookie`. A supplied header replaces, and it lands after the per-request headers, so one value overrides every value the provider chose.

`Content-Type` is picked per request: `application/json`, `application/merge-patch+json`, `application/xml` on some Jamf Pro requests, `multipart/form-data` with a generated boundary for a file upload. One value here overrides all four. A package upload loses the boundary its body was written with, and an XML write is refused in terms that name the resource, not the header.

`Accept` carries `application/xml` on those same Jamf Pro requests. Setting it is no safer on namespaces where the provider sends none. Jamf Security Cloud's UEM Connect service answers `Accept: application/xml` with an XML body, which the provider then tries to decode as JSON, while sibling namespaces answer `406` and fail outright.

A proxy needing a different media type on its own hop has to rewrite the header there. Do not send it to Jamf.

## Troubleshooting

| Symptom | Cause |
|---|---|
| `401` with credentials known good | The proxy is consuming `Authorization`. Set `authorization_header_name`. |
| `403 OWNERSHIP_FORBIDDEN` | Scope mismatch. Check `environment_id` / `tenant_id`, not these attributes. |
| Rejected before Jamf sees it | Missing or misspelled header. `TF_LOG=INFO` lists what is sent. |
| A correct-looking value rejected | Trailing newline from a file or secret store. The provider names the header. |
