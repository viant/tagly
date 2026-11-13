# Boolean enum representation

Integer flags can opt into a native JSON boolean input and output representation:

```go
ManualAlloc int `choice:"0:false" choice:"1:true" enumAccept:"boolean" enumStyle:"boolean" enumNull:"reject"`
```

Either boolean side requires integer storage and exactly the explicit declarations
`0:false` and `1:true`, in either order. Named integer types and pointers are
supported; custom JSON/Text owners retain their existing ownership restrictions.
Other mappings, implicit ordinals, string or boolean storage, and extra choices
fail plan construction. Storage remains integer 0/1.

`DecodeJSON` accepts only JSON `false` and `true`, producing storage JSON `0` and
`1`. Numbers, quoted booleans, arrays and objects are rejected. Null follows the
existing enum policy: it passes through by default; `enumNull:"reject"` preserves
`ErrNullInput` for `errors.Is`. Omission and presence remain the containing
field decoder's responsibility. `Encode` returns native booleans for stored
integer 0/1. Other stored values fail with an error wrapping `ErrUnknownOutput`;
nil output remains nil.

Input and output styles are independent. For example boolean input with ordinal
output preserves integer output, while scalar input with boolean output retains
scalar aliases. Existing scalar-only input policies and string-only output alias
restrictions still apply; boolean input does not accept aliases or fallbacks.

For the selected boolean side, `WireType` returns `"boolean"` and `WireValues`
returns native booleans in declaration order. `StorageValues` retains typed
integer values. `InputDomain` and `OutputDomain` remain nil for a single boolean
representation; schema consumers use the existing wire accessors and
`AllowsNullInput`. Returned wire value slices are independent of the compiled
plan. Conversion reuses the existing enum plan and JSON decoder without an
additional cache, parser or request-time declaration scan. JSONx is unchanged.
