# Native JSON input policies

Consumers explicitly compile `jsonx` tags and apply them before native typed
decoding and presence tracking. These tags do not change `encoding/json` or
output serialization.

`jsonx:"wrapper=ids"` admits either a bare integer array or an object containing
that array in the declared member, for example `[1,2]` or `{"ids":[1,2]}`.
The member name is configuration; it is not inferred from a field or route name.
The initial supported destination is a direct slice of native `int` values,
including method-free named types. Custom JSON/Text owners are rejected.

Wrapped objects require a nonnull member and reject unknown members. Member
matching is case-insensitive, following Go JSON struct decoding. Duplicate
members retain their final value, but an invalid earlier value still rejects
the input. Empty arrays remain valid and nonnil. Bare `null` remains
decoder-owned (or is omitted with `null=omit`); wrapped null is invalid.

Unknown wrapper members and missing/null wrapper values return `*WrapperError`.
Its `PublicMessage` method identifies only these native validation messages;
nil and zero-value errors return no public message. Consumers must check the
concrete type and preserve their own status and binding-owner rules. Other
syntax, type, conversion and configuration errors are not classified as public.

The policy normalizes the envelope's integer array for the consumer's existing
decoding pipeline, including Go's slice reuse for duplicate members and null
integer elements. `InputDomain.Wrapper` identifies this object envelope independently
of `InputDomain.Object`, which describes the record admitted by `objectToSlice`.
Schema consumers must place the destination array schema inside a required,
nonnull wrapper property and disallow additional properties. Policies apply to
input only; output remains the destination's ordinary representation.
