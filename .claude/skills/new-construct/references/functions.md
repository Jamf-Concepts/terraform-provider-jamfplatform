# Provider-defined functions

Authoritative: `STYLE_GUIDE.md` §Provider Function File Conventions and §Testing a provider
function. Functions are **offline** — called before provider configuration is evaluated, so no
SDK client, no credentials, no provider config. Namespace: `jamfplatform::<fn>`, package
`internal/functions/<fn>/`.

## Shape

`function.go` holds `Metadata`, `Definition` and `Run` plus the argument decode; a separate
`<core>.go` holds the framework-free core returning `([]byte, error)` or a plain value. `Run`
decodes, delegates, sets the result — nothing else.

Declare arguments as `function.DynamicParameter` and decode with `req.Arguments.Get` →
`helpers.TerraformDynamicToJSON` → `map[string]any`. `DynamicParameter` rather than a typed
`ObjectParameter`/`ListParameter` is deliberate: it lets a heterogeneous input — a list whose
objects have different key sets — decode as a cty tuple instead of being rejected for
non-uniform element types. Guard the top-level type assert and return
`function.NewArgumentFuncError(argIndex, …)` on mismatch.

Share a core between related functions rather than duplicating it (`mcx_forced_payload` imports
`mobileconfig` for `Assemble`). At a third consumer, lift the neutral core into its own package
that each function imports, rather than importing one function package from another.

Register via the provider's `Functions()` method.

## Testing both sides of the seam

- **Core unit tests** feed Go values straight to the core — fast, exhaustive, and where golden
  output is pinned.
- **`Run` seam tests** build a real `types.Dynamic` and call `Run` through
  `function.NewArgumentsData([]attr.Value{…})`, asserting the happy path **and** at least one
  argument-error path. This is the path Terraform actually invokes; core tests bypass it.
- **Acceptance** invokes the function from a real config through an `output` and asserts with
  `statecheck.ExpectKnownOutputValue` + `knownvalue.StringRegexp`. It **must not** call
  `testhelpers.AccPreCheck` — that gates on tenant credentials and would skip. Call
  `testhelpers.AccPreCheckOffline(t)`, use `ProtoV6ProviderFactories` directly, and gate only
  on `tfversion.SkipBelow(tfversion.Version1_8_0)`.
