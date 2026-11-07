// Package jsonx compiles opt-in native JSON input conversion metadata.
// Consumers must explicitly own decoding; these tags do not change encoding/json.
package jsonx

import (
	"bytes"
	"encoding"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// InputDomain describes additional admitted input representations. A true String
// or Integer denotes an unrestricted branch, subject to any exact bounds.
// Arrays retain the destination's item schema. Null is always decoder-owned.
type InputDomain struct {
	Boolean, String, Integer, Number, Array, Null bool
	IntegerValues                                 []json.Number
	StringValues                                  []string
	Minimum, Maximum                              *json.Number
}

// Policy is immutable after compilation and safe for concurrent use.
type Policy struct {
	conversion string
	omitNull   bool
}

// Compile preserves unknown jsonx options for compatibility with other owners.
// It rejects incompatible known conversions and unsupported destination kinds.
func Compile(tag reflect.StructTag, typ reflect.Type) (*Policy, error) {
	p := &Policy{}
	for _, option := range strings.Split(tag.Get("jsonx"), ",") {
		option = strings.TrimSpace(option)
		switch option {
		case "null=omit":
			p.omitNull = true
		case "boolToInt", "intToBool", "numberToString", "commaStringToSlice", "boolIntToYN":
			if p.conversion != "" && p.conversion != option {
				return nil, fmt.Errorf("incompatible jsonx conversions %s and %s", p.conversion, option)
			}
			p.conversion = option
		}
	}
	if !p.Active() {
		return p, nil
	}
	if typ == nil {
		return nil, fmt.Errorf("jsonx requires destination type")
	}
	jsonOwner := reflect.TypeOf((*json.Unmarshaler)(nil)).Elem()
	textOwner := reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()
	for {
		if typ.Implements(jsonOwner) || typ.Implements(textOwner) {
			return nil, fmt.Errorf("jsonx cannot override custom JSON/Text ownership on %v", typ)
		}
		if typ.Kind() != reflect.Pointer {
			break
		}
		typ = typ.Elem()
	}
	if reflect.PointerTo(typ).Implements(jsonOwner) || reflect.PointerTo(typ).Implements(textOwner) {
		return nil, fmt.Errorf("jsonx cannot override custom JSON/Text ownership on %v", typ)
	}
	valid := true
	switch p.conversion {
	case "boolToInt":
		valid = typ.Kind() == reflect.Int
	case "intToBool":
		valid = typ.Kind() == reflect.Bool
	case "numberToString", "boolIntToYN":
		valid = typ.Kind() == reflect.String
	case "commaStringToSlice":
		valid = typ.Kind() == reflect.Slice && typ.Elem().Kind() == reflect.String
	}
	if !valid {
		return nil, fmt.Errorf("jsonx %s unsupported destination %v", p.conversion, typ)
	}
	return p, nil
}
func (p *Policy) Active() bool { return p != nil && (p.omitNull || p.conversion != "") }

// Apply converts one admitted field value, before typed assignment/presence.
// An omitted null never reaches the destination; a conversion-produced null does.
func (p *Policy) Apply(raw []byte) ([]byte, bool, error) {
	if !p.Active() {
		return raw, false, nil
	}
	token := bytes.TrimSpace(raw)
	if bytes.Equal(token, []byte("null")) {
		return raw, p.omitNull, nil
	}
	fail := func() ([]byte, bool, error) {
		return nil, false, fmt.Errorf("invalid JSON input for jsonx %s", p.conversion)
	}
	switch p.conversion {
	case "boolToInt":
		if bytes.Equal(token, []byte("true")) {
			return []byte("1"), false, nil
		}
		if bytes.Equal(token, []byte("false")) {
			return []byte("0"), false, nil
		}
	case "intToBool":
		if bytes.Equal(token, []byte("true")) || bytes.Equal(token, []byte("false")) {
			return raw, false, nil
		}
		n, err := nativeInteger(token)
		if err != nil || n != 0 && n != 1 {
			return fail()
		}
		return []byte(strconv.FormatBool(n == 1)), false, nil
	case "numberToString":
		var text string
		if json.Unmarshal(token, &text) == nil {
			return raw, false, nil
		}
		if !json.Valid(token) || len(token) == 0 || token[0] != '-' && (token[0] < '0' || token[0] > '9') {
			return fail()
		}
		encoded, _ := json.Marshal(string(token))
		return encoded, false, nil
	case "commaStringToSlice":
		if len(token) > 0 && token[0] == '[' {
			var values []string
			if json.Unmarshal(token, &values) != nil {
				return fail()
			}
			return raw, false, nil
		}
		var text string
		if json.Unmarshal(token, &text) != nil {
			return fail()
		}
		var values []string
		for _, cell := range strings.Split(text, ",") {
			if cell = strings.TrimSpace(cell); cell != "" {
				values = append(values, cell)
			}
		}
		encoded, _ := json.Marshal(values)
		return encoded, false, nil
	case "boolIntToYN":
		value := ""
		switch string(token) {
		case "true":
			value = "Y"
		case "false":
			value = "N"
		default:
			if n, err := nativeInteger(token); err == nil {
				if n == 0 {
					value = "N"
				} else {
					value = "Y"
				}
			} else {
				var text string
				if json.Unmarshal(token, &text) != nil {
					return fail()
				}
				switch strings.ToUpper(text) {
				case "Y", "1":
					value = "Y"
				case "N", "0":
					value = "N"
				case "":
				default:
					return fail()
				}
			}
		}
		encoded, _ := json.Marshal(value)
		return encoded, false, nil
	}
	return raw, false, nil
}
func nativeInteger(token []byte) (int64, error) {
	if !json.Valid(token) || bytes.ContainsAny(token, ".eE") {
		return 0, fmt.Errorf("integer token required")
	}
	return strconv.ParseInt(string(token), 10, strconv.IntSize)
}

// InputDomain returns independent metadata; callers may safely modify it.
// A policy with only null=omit leaves the destination's ordinary domain intact.
func (p *Policy) InputDomain() InputDomain {
	d := InputDomain{Null: true}
	if p == nil {
		return d
	}
	switch p.conversion {
	case "boolToInt":
		d.Boolean = true
		d.Integer = true
		d.Minimum, d.Maximum = nativeBounds()
	case "intToBool":
		d.Boolean = true
		d.IntegerValues = []json.Number{"0", "1"}
	case "numberToString":
		d.String = true
		d.Number = true
	case "commaStringToSlice":
		d.String = true
		d.Array = true
	case "boolIntToYN":
		d.Boolean = true
		d.Integer = true
		d.StringValues = []string{"Y", "y", "N", "n", "0", "1", ""}
		d.Minimum, d.Maximum = nativeBounds()
	}
	return d
}
func nativeBounds() (*json.Number, *json.Number) {
	min, max := json.Number("-2147483648"), json.Number("2147483647")
	if strconv.IntSize == 64 {
		min, max = "-9223372036854775808", "9223372036854775807"
	}
	return &min, &max
}
