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

## Explicit open integer policies

Integer storage with ordinal output can opt into `enumInteger:"passthrough"`
and/or `enumUnknownString:"0"`. Each is independent and requires scalar input.
The former accepts every JSON integer fitting the actual Go storage width; the
latter maps any otherwise unknown string to the specified storage-valid integer,
which need not be a declared choice. Neither trims strings nor parses quoted
numbers. Null remains decoder-owned. Defaults remain strict.

`enumCase:"upper"` compares `strings.ToUpper` results, including its Unicode
mapping (for example dotless i). It is distinct from `fold`. Alias collisions
are rejected under the selected comparison. Known names still take precedence
over the unknown-string fallback.

`InputDomain.OpenIntegers` and `OpenStrings` describe unrestricted branches.
Consumers must not apply the finite choice lists to those branches. Integer
bounds retain exact decimal tokens in `Minimum` and `Maximum`; consumers using
floating-point schema bounds should emit only exactly representable bounds.
Open ordinal output returns no finite `WireValues(false)` restriction. An absent
finite enum means unrestricted storage values, never an enum containing only null.

## Canonical string fallbacks

String storage with literal output can independently map unknown strings or
unknown integers to a declared choice:

```go
Mode *string `json:"mode" choice:"0:ASAP" choice:"1:SPEND_EVENLY" enumAccept:"scalar" enumStyle:"literal" enumCase:"fold" enumUnknownString:"0" enumUnknownInteger:"0"`
```

Known ordinals, names and explicit aliases take precedence. Each fallback tag
names a declared ordinal whose canonical literal is stored and emitted. Here
numeric `1` and case variants of `SPEND_EVENLY` become `SPEND_EVENLY`; other
integers and strings become `ASAP`. Quoted `"1"` and `"Even"` are unknown strings
unless explicitly authored as aliases. Case matching, aliases and null behavior
remain independent. Without a fallback, unknown inputs retain strict rejection.

`enumUnknownInteger` accepts exactly the signed native Go `int` range, even
though storage is string. Known choice ordinals must also fit that range. Fractions,
exponent spellings and overflow fail before fallback. For string storage the tag requires scalar
input and literal output; it cannot combine with integer passthrough. String-storage `enumUnknownString` also requires literal output and
a declared target. Existing integer-storage unknown-string policies are unchanged.

`InputDomain` exposes open branches and exact decimal `Minimum`/`Maximum` bounds
for the native-int input domain. Preserve these numbers exactly in serialized
schemas: even binary-exact float64 bounds can serialize to a different decimal
integer. Literal output still advertises only canonical choices. JSON Schema's
mathematical integer type cannot express the decoder's integer-token spelling
restriction (`1.0` and `1e0` remain runtime errors).


Integer storage with ordinal output can use the same `enumUnknownInteger`
policy to normalize unknown integers to a declared ordinal:

```go
Mode *int `json:"mode" choice:"1:FIXED_BUDGET" choice:"2:AUTO_BUDGET" enumAccept:"scalar" enumCase:"upper" enumUnknownString:"1" enumUnknownInteger:"1"`
```

Numeric `2` and the named `AUTO_BUDGET` value become `2`; other accepted integers
and unknown strings become `1`. Input integers must fit the actual Go storage
width before fallback; signedness and overflow checks are unchanged. Integer
fallback and passthrough are mutually exclusive. Output choices remain finite
when all fallbacks target declared choices. The pre-existing integer-storage
unknown-string policy may still target an undeclared storage-valid integer;
that combination retains unrestricted ordinal output metadata. Without the new
integer fallback tag, all previous ordinal policy behavior remains unchanged.
