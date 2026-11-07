# Native input source groups

Fields may explicitly share a source group when distinct wire members represent
one application value but require different typed interpretation:

```go
Canonical *int16 `json:"day_of_week" format:"inputGroup=weekday,inputPriority=0,inputNames=dayOfWeek,inputNamesPriority=alias"`
Upper *int `json:"Weekday" format:"inputGroup=weekday,inputPriority=1,inputCase=exact"`
Lower *int16 `json:"weekday" format:"inputGroup=weekday,inputPriority=2,inputCase=exact"`
```

`inputGroup` requires a nonempty trimmed name and an explicit `inputPriority`.
Priority is a nonnegative decimal uint32; larger values win. Priorities within
one visible group must be unique. A singleton group is allowed, including when
other members were hidden. Priority without a group is invalid. Group names
are local to the declaring object's selected field plan.

`inputCase=exact` disables canonical case folding for that field. Default
canonical matching is unchanged. Alias case matching remains independently
controlled by `inputNamesCase`. Exact canonical ownership precedes aliases.

The existing matcher first applies per-field alias admission, kind restrictions
and alias/canonical precedence. It then selects the highest-priority admitted
present field in each group. Null counts as present. Every occurrence of a
losing field is discarded before conversion, typed assignment and presence.
Winning duplicates retain native order and earlier malformed winning values
still fail. Invalid JSON syntax never becomes acceptable through suppression.
Unknown names retain the decoder's unknown-field behavior.

Consumers must use the same compiled matcher for values and presence. Hidden
fields cannot win. This metadata performs no arithmetic: application projection
converts typed source members into the business model. It does not change output
or generic encoding/json behavior. Existing Tagly InputName remains a two-field
struct; provide policies through NewInputNameMatcher's existing variadic argument.

Schemas must not require every source or validate a malformed losing source.
Native schema owners can expose explicit conservative source-group metadata and
leave selection, winning-value validation and group requiredness to runtime.
Do not activate this projection for generic decoders or output schemas.

## Exact exclusions of canonical case variants

`json:"Status" format:"inputNames=status,inputNamesExclude=status"` ignores only
exact lowercase status while retaining Status, STATUS and other canonical-folded
spellings. The excluded spelling must be a declared alias and must differ from
its own exact canonical name. Exact canonical spelling and other fields' canonical
ownership remain protected. Resolution order is exact canonical, exact exclusions,
folded canonical, then aliases. No exclusion affects unconfigured fields.

## Ancestor presence guards

`format:"inputUnless=../Canonical"` suppresses the annotated field when the
visible Go field `Canonical` is admitted in the enclosing struct object.
Additional `../` segments select further enclosing struct objects; pointers,
map values, and slice elements do not add levels. Targets use exported Go names,
not JSON aliases. Invalid syntax, unavailable depths, and hidden or missing
ancestor targets are configuration errors even for an empty input.

Presence is determined by the existing native field-selection plan before value
conversion: an explicit null counts, including a null subsequently omitted by a
conversion policy. Excluded aliases, aliases with ignored shapes, and losing
input-group members do not count. Guarded fields are removed before their own
alias/group selection, conversion, narrowing, and presence marking. The remaining
active occurrences retain existing deterministic duplicate and syntax validation.
No guard is implied for untagged fields.

Owners compile `InputScopePlan` alongside their ordinary field plans and validate
it with `ValidateInputScopePlan`; runtime selection uses `SelectScoped` with
nearest-first `InputScope` values. Recursive contexts are validated, and schema
owners may conservatively admit guarded values while documenting that native
runtime selection determines which values undergo validation. Generic JSON and
output schemas must not silently adopt native input policies.
