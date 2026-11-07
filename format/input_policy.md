# Input alias admission

`format:"inputNames=oldName"` remains an exact, decode-only alias. It keeps the
native decoder's lexical duplicate behavior and ordinary typed errors.

Four independent, opt-in options refine existing alias admission:

| Option | Value | Effect |
| --- | --- | --- |
| `inputNamesPriority` | `canonical` | Ignore aliases when any canonical property is supplied. |
| `inputNamesKinds` | Pipe-separated JSON kinds | Ignore alias values of other kinds. |
| `inputNamesCase` | `fold` | Match alias names with `strings.EqualFold`. |
| `inputNamesExclude` | Pipe-separated exact names | Ignore these alias spellings, before case folding. |

Every option requires `inputNames`. JSON kinds are `object`, `array`, `string`,
`number`, `boolean`, and `null`. Invalid options, ambiguous aliases, exclusions
that do not match the field's aliases, and exclusions colliding with canonical
fields fail during plan construction. Exclusions never hide canonical fields.

```go
Cap *Constraint `json:"frequencyCapping" format:"inputNames=frequencyCap,inputNamesPriority=canonical,inputNamesKinds=object|null,inputNamesCase=fold"`
Flights []Flight `json:"campaignFlightCaps" format:"inputNames=flightConstraints,inputNamesPriority=canonical,inputNamesCase=fold,inputNamesExclude=FlightConstraints"`
```

In the first example, a supplied canonical `frequencyCapping` always wins over
its alias, including when the canonical value is null or invalid. An alias
object or null is decoded normally when canonical input is absent; scalar and
array alias values are ignored without setting presence. Invalid canonical
values and invalid content within admitted alias objects remain typed errors.
Malformed JSON remains a syntax error even in an ignored value.

The second example accepts folded forms of `flightConstraints` except the exact
spelling `FlightConstraints`. Only canonical names appear in output. Within the
winning representation, existing duplicate assignment and strict duplicate
rejection rules still apply. These options do not replace native duplicates
with map iteration or last-only decoding.

`NewInputNameMatcher` accepts optional per-field `*Tag` policies, preserving its
existing callers and the two-field `InputName` shape. `Classify` identifies
canonical, alias and excluded spellings; `Select` receives already-parsed
`InputMember` occurrences and selects admitted members without losing order.
The caller retains JSON parsing, visibility, typed conversion and presence.

Consumers must project these policies into recursive input discovery before
advertising them. Schemas must accept ignored alias kinds, preserve meaningful
constraints on admitted objects, and account for canonical precedence and
requiredness. Output schemas remain canonical. Structology and Bindly own codec
and presence behavior; schema exposure belongs to the HTTP/MCP schema owner.
