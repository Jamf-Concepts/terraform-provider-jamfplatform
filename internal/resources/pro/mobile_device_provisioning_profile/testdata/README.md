# Acceptance test fixtures

`Certificates.mobileprovision` is a real, Apple-signed enterprise `.mobileprovision`
profile used to exercise the create-with-blob path. Jamf Pro cryptographically
validates the profile on upload and **rejects synthesised / self-signed blobs with
HTTP 500**, so a genuine signed profile is required — a generated fixture will not
work.

Provenance: sourced from the public repository
<https://github.com/loyahdev/certificates> (file `Certificates.mobileprovision`).
It contains only the certificates already published there; nothing tenant- or
account-specific is added by this repo.

The acceptance test base64-encodes this file at runtime and feeds it to
`profile_data`. The embedded profile carries a fixed UUID and expiration that the
server parses back out — the tests assert those computed fields are populated, not
their exact values (the profile may expire).

## ⚠️ Expiration fuse

This fixture's embedded `ExpirationDate` is **2026-06-26** (1-year enterprise
profile, created 2025-06-26). Uploads of a non-expired profile succeed (verified).
If Jamf Pro rejects an *expired* profile at upload — unverified — the acceptance
`create` steps will begin failing **after 2026-06-26**. When that happens, replace
this file with a longer-lived signed `.mobileprovision` (any real enterprise
profile works; synthesised/self-signed blobs are rejected with HTTP 500). All
real enterprise profiles expire, so this fixture will need periodic refresh.

Decode the dates with:

```bash
openssl smime -verify -inform DER -noverify -in Certificates.mobileprovision 2>/dev/null \
  | plutil -extract ExpirationDate raw -
```
