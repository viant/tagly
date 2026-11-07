// Package format defines metadata for formatting struct field values.
//
// The inputNames option declares pipe-separated, exact-match decode-only names:
//
//	Metric25 bool `json:"metric25,omitempty" format:"inputNames=metric_25"`
//
// These names do not replace the canonical JSON output name or apply a case
// conversion. Empty and duplicate aliases are invalid. Decoders must explicitly
// consume this metadata, preserve field visibility and reject aliases that
// ambiguously address multiple fields; encoding/json itself does not read it.
package format
