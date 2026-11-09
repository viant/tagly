package choice

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
)

// ScalarValue retains the branch of a JSON string-or-integer value. The zero
// value represents null; exactly one non-nil field represents a present scalar.
// It is native representation storage, not a wire DTO. It deliberately has no
// JSON or text marshaling methods: representation metadata owns wire conversion.
// In particular, a numeric catalog ID must not become indistinguishable from a
// string alias before domain validation.
type ScalarValue struct {
	String  *string `json:"string"`
	Integer *int64  `json:"integer"`
}

// ParseScalarValue preserves strings verbatim and integers exactly within int64.
// Catalog normalization, aliases, empty/zero policies and validation belong to
// the selected representation plan. Null does not imply request omission;
// presence remains owned by the containing decoder.
func ParseScalarValue(raw []byte) (ScalarValue, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return ScalarValue{}, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err != nil {
			return ScalarValue{}, err
		}
		return ScalarValue{}, fmt.Errorf("unexpected trailing scalar JSON value")
	}
	switch actual := value.(type) {
	case nil:
		return ScalarValue{}, nil
	case string:
		return ScalarValue{String: &actual}, nil
	case json.Number:
		number, err := strconv.ParseInt(string(actual), 10, 64)
		if err != nil {
			return ScalarValue{}, fmt.Errorf("scalar requires an int64 JSON integer: %w", err)
		}
		return ScalarValue{Integer: &number}, nil
	default:
		return ScalarValue{}, fmt.Errorf("scalar requires a JSON string, integer or null")
	}
}

// Value returns the original scalar without normalizing its branch. Invalid
// programmatic storage is rejected rather than silently choosing one branch.
func (s ScalarValue) Value() (any, error) {
	if s.String != nil && s.Integer != nil {
		return nil, fmt.Errorf("scalar has both string and integer branches")
	}
	if s.String != nil {
		return *s.String, nil
	}
	if s.Integer != nil {
		return *s.Integer, nil
	}
	return nil, nil
}
