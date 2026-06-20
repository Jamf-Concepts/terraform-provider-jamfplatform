# testdata

## `signing.p12`

A **dummy self-signed PKCS#12 keystore** used exclusively by the acceptance
tests in this package (`resource_acceptance_test.go`). It is **not** a real
signing certificate and carries no security value.

- Password: `TestPass123`
- Self-signed, no chain, throwaway key.
- Used only to exercise the keystore-upload code paths
  (`mdm_signing_certificate` and `developer_certificate`). The MDM-signing
  slot ingests this dummy cert and populates `subject`; the developer-cert
  slot may reject it (Jamf expects an Apple Developer ID cert), so the
  developer-cert acceptance test asserts only that the upload does not error.

The keystore field is `WriteOnly`, so it cannot be referenced as a file path in
HCL (acceptance configs run from a Terraform temp dir). The test reads this file
with `os.ReadFile("testdata/signing.p12")`, base64-encodes it, and injects the
string directly into `keystore_file`.

### Regeneration

```sh
openssl req -x509 -newkey rsa:2048 -keyout key.pem -out cert.pem \
  -days 3650 -nodes -subj "/CN=tf-acc-dummy-signing"
openssl pkcs12 -export -inkey key.pem -in cert.pem \
  -out signing.p12 -passout pass:TestPass123
rm key.pem cert.pem
```
