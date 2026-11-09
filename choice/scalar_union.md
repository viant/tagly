# Branch-preserving scalar storage

`ScalarValue` stores a string or signed 64-bit integer without custom JSON or
text methods. Use it when normalization must retain the input branch for later
business validation. Ordinary integer and string enums remain unchanged.

```go
Fee choice.ScalarValue `json:"fee" choice:"4237:AD_SERVING" choice:"4241:AD_SERVING" enumAccept:"scalar" enumStyle:"scalar" enumAlias:"4237:Ad Serving" enumCase:"fold" enumTrim:"space" enumInteger:"passthrough" enumUnknownString:"preserve" enumEmpty:"null" enumZero:"null"`
```

`ParseEnum` compiles these tags. `DecodeJSON` returns JSON for native storage;
the containing decoder retains request presence. `Encode` returns the public
scalar. Standard JSON encoding of `ScalarValue` itself is its storage shape,
**not** the public representation. Consumers must use the compiled native plan.

- Integer inputs retain their exact ID, even when several IDs share a literal.
- String inputs normalize through the existing alias, case and trim policies.
- Known integers emit their canonical literal. With `enumInteger:"passthrough"`,
  unknown integers emit unchanged; otherwise they fail.
- `enumUnknownString:"preserve"` retains unknown strings for business validation.
  Without it, unknown strings fail. This is not a fallback to a known enum.
- `enumEmpty:"null"` and `enumZero:"null"` independently select empty-string and
  zero normalization. Explicit null is always accepted; omission is not inferred.
- Fractional/exponent numeric spellings, overflow, non-scalar JSON, multiple
  JSON values and storage with both branches set fail.

This mode requires `enumAccept:"scalar"` and `enumStyle:"scalar"`. Duplicate
ordinals and aliases addressing different canonical literals fail compilation.
Multiple ordinals sharing exactly the same literal are permitted. Catalog and
domain/persistence decisions remain in application code.

Schema consumers must handle `InputDomain()` and `OutputDomain()`, including
`Nullable` and integer bounds, before using `WireType`/`WireValues`. A union has
no single `WireType`. `StorageValues` is not a finite validation domain for this
representation. Runtime and schema consumer integration must be verified before
applications adopt the tags; these metadata APIs alone do not establish support
in Bindly, Structology, HTTP or MCP.

With integer passthrough, the output schema's bounded integer branch is a
conservative superset: declared integers actually emit canonical strings, but
the metadata does not enumerate integer exclusions. Consumers requiring an
exact output domain must account for this limitation.
