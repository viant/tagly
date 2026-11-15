# Exact literal string input

Scalar string storage can explicitly require canonical named input:

```go
Mode *string `choice:"0:ASAP" choice:"1:SPEND_EVENLY" enumAccept:"literal" enumUnknownString:"reject"`
```

This admits exactly `ASAP` and `SPEND_EVENLY`. Unknown strings and empty strings
are rejected unless declared. JSON numbers, booleans, arrays, and objects remain
invalid literal inputs. Null passes through unless `enumNull:"reject"` applies;
omission remains owned by the containing field.

Without this explicit policy, literal string input retains its permissive
passthrough behavior. The policy requires explicit literal input and scalar
string storage; it does not enable aliases, case conversion, or trimming.
Scalar-mode numeric fallbacks and collection policies retain their existing
semantics. Custom JSON/Text owners cannot be overridden.

Canonical membership is indexed once in the immutable compiled enum plan using
the same indexing machinery as string collections. Decoding performs an exact
map lookup. Output aliases are excluded from input membership, and output
behavior remains independently controlled by the existing output policies.
