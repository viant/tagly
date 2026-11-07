# Scalar enum input

An opt-in scalar input accepts declared integer IDs and declared string names
while retaining one canonical storage and output representation:

```go
Mode *string `json:"mode" choice:"0:ASAP" choice:"1:SPEND_EVENLY" enumAccept:"scalar" enumStyle:"literal" enumAlias:"0:0" enumAlias:"1:1" enumAlias:"1:Even" enumCase:"fold"`
```

Here `0`, `"0"`, and case variants of `"ASAP"` store `"ASAP"`.
`1`, `"1"`, and case variants of `"Even"` or `"SPEND_EVENLY"` store
`"SPEND_EVENLY"`. Output uses canonical names; aliases do not become output
choices. Without explicit numeric string aliases, `"0"` and `"1"` are rejected.

Each `enumAlias:"ordinal:string"` names an existing ordinal. Only the first
colon separates the ordinal from the string. Aliases and `enumCase:"fold"`
require scalar input. Without case folding, names match exactly. Folding uses
Unicode simple folding, matching Go's `strings.EqualFold`. Ambiguous names or
aliases referring to different ordinals are rejected during plan construction.

Scalar input rejects unknown values, booleans, arrays and objects. Numbers must
have an integer token spelling: `1.0` and `1e0` are rejected. Conversion uses
`json.Number`, preserving large integer identity. Null passes through to the
decoder; pointer allocation and presence tracking remain decoder responsibilities.
Output styles remain `ordinal` and `literal`; scalar is input-only. Existing
ordinal/literal input modes retain their behavior, including permissive
same-representation values.

## Input schema consumers

Use `Enum.InputDomain()` for scalar inputs. It returns integer values, string
tokens and string alternatives. Describe the input using an integer enum
branch OR one branch per string alternative, adding nullability at the owning boundary.
Do not combine a case-sensitive string enum with the folded pattern. The pattern
includes Unicode fold equivalents. Each string branch must emit `pattern` plus
both `minLength` and `maxLength` equal to the alternative's `Length`, including
zero. These lengths are Unicode rune counts. Exact length prevents JavaScript's
end anchor from accepting before an extra final newline, while avoiding
lookahead syntax that Go's RE2-based schema validators reject. Supplementary-plane
case-fold matching should be verified against the client's JavaScript regex mode.

`WireType(true)` returns an empty string and `WireValues(true)` returns nil for
scalar input because neither legacy accessor can represent that union. Existing
input modes and output accessors are unchanged. Schema consumers must support
the structured input domain before advertising scalar-tagged fields.

JSON Schema integer validation is mathematical and cannot distinguish `1` from
`1.0` or `1e0`; the decoder's numeric lexical restriction remains stricter.

## Source compatibility

`Entry` retains its two-field shape. `Enum` includes private compiled alias
state; external code should obtain plans through `ParseEnum`, or use keyed
construction for its exported fields. External unkeyed `Enum` literals require
updating when adopting this version.
