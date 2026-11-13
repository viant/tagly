# Output-only enum aliases

String storage with literal output can normalize explicitly declared stored
tokens to a canonical output literal without changing input conversion:

```go
State string `choice:"0:Pending" choice:"10:Active" choice:"20:Unsupported" enumAccept:"scalar" enumStyle:"literal" enumOutputAlias:"0:pending-token" enumOutputAlias:"10:active-token" enumOutputAlias:"20:unknown"`
```

`Encode("active-token")` emits `"Active"`. `Encode("unknown")` emits
`"Unsupported"` because that specific stored token is declared. Canonical
literals emit unchanged. Other stored strings pass through unchanged by default;
they do not implicitly map to the declared unsupported choice. Nil remains nil.
Field omission remains owned by the containing encoder.

Each repeated `enumOutputAlias:"ordinal:stored-token"` references a declared
choice ordinal. Only the first colon separates the ordinal and stored token;
tokens may contain colons or be empty. Matching is exact, including case and
whitespace. The existing `enumAlias`, `enumCase`, and `enumTrim` policies apply
only to input and never alter output token matching. Output aliases do not join
the accepted input domain. Existing permissive literal input stays permissive.

The native tag parser and ordinal parser compile an immutable exact token lookup
once per enum plan. Malformed declarations, undeclared ordinals, duplicate alias
tokens, aliases colliding with canonical literals, and incompatible storage or
output styles fail plan construction. Integer storage and `ScalarValue` cannot
use these output policies. Custom JSON/Text owners retain their contract.

## Unknown outputs and schema metadata

The default unknown-token passthrough requires an open output string schema:
`WireType(false)` returns `"string"`, `WireValues(false)` returns nil, and
`OutputDomain()` remains nil. A schema consumer must omit a finite enum when
`WireValues` is nil. It must not advertise only the canonical choice list or
interpret nil as an enum containing null. Input schema metadata is unchanged.

Add `enumOutputUnknown:"reject"` to reject stored strings outside the canonical
literals and output aliases. `Encode` returns an error preserving
`ErrUnknownOutput` for `errors.Is`. This opt-in supports a finite output schema:
`WireValues(false)` returns only the canonical literals in declaration order.
The rejection policy also works without output aliases. Its only accepted tag
value is `reject`; omitting it selects passthrough. Duplicate policy declarations
fail plan construction.

These policies are opt-in; existing enum output behavior is unchanged without
either output tag. Input decoding, input fallbacks, null admission, and presence
tracking retain their existing owners.
