# Native JSON input policy

`Compile(tag, destinationType)` compiles explicit `jsonx` metadata once. `Apply`
converts one already-admitted member before typed assignment and presence tracking.
Consumers retain field visibility, custom decoding ownership, alias winner
selection, recursive traversal, duplicate policy and error paths. It is not a
replacement JSON decoder, and generic encoding/json does not honor these tags.

Supported conversions are `boolToInt`, `intToBool`, `numberToString`,
`commaStringToSlice`, and `boolIntToYN`; each may combine with `null=omit`.
Only one conversion is permitted. Unknown options retain prior compatibility.
Integer input uses strict integer-token spelling and native signed bounds.
Number-to-string preserves arbitrary numeric lexemes without float conversion.
Comma strings trim/drop empty cells, with an all-empty result represented by
null; arrays are passed through after []string validation. Y/N conversion accepts
booleans, any native signed integer, and strings Y/y/N/n/0/1/empty, without trimming.

`InputDomain` supplies input union branches only. String/Integer true mean open
branches; StringValues/IntegerValues mean finite branches. Minimum/Maximum are
exact json.Number values and must never pass through float64. Array retains the
existing destination's item schema. Null is accepted for these conversions and
omission policy. A domain with no non-null branch represents null=omit only and
retains the destination's ordinary non-null domain. Output retains its ordinary
storage representation. Returned metadata may be mutated without changing policy.

JSON Schema integer means mathematical integer; it cannot distinguish 1 from
1.0 or 1e0. Document this runtime lexical restriction. Consumer schema projection
must remain explicitly tied to its native decoder, including aliases, required
presence and output direction. Do not apply these input domains to generic
encoding/json consumers, which do not perform the conversions.
