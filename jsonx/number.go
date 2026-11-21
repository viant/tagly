package jsonx

import (
	"bytes"
	"encoding/json"
	"io"
	"strconv"
	"strings"
)

// numberAsInt64 follows the permissive numeric domain of the native release decoder.
// Decimal and exponent tokens take the floating-point path. Whole numbers retain
// full precision, including the unsigned range, until the final integer cast.
func numberAsInt64(raw string) (int64, error) {
	if strings.ContainsAny(raw, ".eE") {
		value, err := strconv.ParseFloat(raw, 64)
		return int64(value), err
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err == nil {
		return value, nil
	}
	unsigned, err := strconv.ParseUint(raw, 10, 64)
	return int64(unsigned), err
}

// ValidateDiscardedNumbers preserves the decoder's number range checks even
// when duplicate selection removes an otherwise valid object member.
// raw must contain one syntactically valid JSON value.
func ValidateDiscardedNumbers(raw []byte) error {
	// A valid scalar has no nested numbers. Check its range directly and
	// reserve the token walk for objects and arrays.
	raw = bytes.TrimSpace(raw)
	if len(raw) > 0 && raw[0] != '{' && raw[0] != '[' && json.Valid(raw) {
		switch raw[0] {
		case '"', 't', 'f', 'n':
			return nil
		default:
			_, err := numberAsInt64(string(raw))
			return err
		}
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	for {
		token, err := d.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if n, ok := token.(json.Number); ok {
			if _, err = numberAsInt64(n.String()); err != nil {
				return err
			}
		}
	}
}
