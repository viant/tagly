// Package choice owns ordered, typed scalar metadata declared by repeated
// Go struct choice tags. It does not validate runtime input values.
package choice

import (
	"encoding/json"
	"fmt"
	"github.com/viant/tagly/tags"
	"math"
	"reflect"
	"strconv"
	"strings"
)

type Constraint struct {
	Type   reflect.Type
	Values []interface{}
}

// WireValues returns parsed choices as built-in scalars for schema metadata.
// It preserves order and exact integers while removing named-type marshaler
// methods. Values retains the declared Go type for runtime membership checks.
func (c *Constraint) WireValues() []interface{} {
	result := make([]interface{}, 0, len(c.Values))
	for _, value := range c.Values {
		v := reflect.ValueOf(value)
		var plain interface{}
		switch v.Kind() {
		case reflect.String:
			plain = v.String()
		case reflect.Bool:
			plain = v.Bool()
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			plain = v.Int()
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			plain = v.Uint()
		case reflect.Float32:
			plain = float32(v.Float())
		case reflect.Float64:
			plain = v.Float()
		}
		result = append(result, plain)
	}
	return result
}

func (c *Constraint) JSONType() string {
	switch c.Type.Kind() {
	case reflect.String:
		return "string"
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "integer"
	default:
		return "number"
	}
}

func scalar(t reflect.Type) (reflect.Type, error) {
	if t == nil {
		return nil, fmt.Errorf("choice requires an explicit scalar type")
	}
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.String, reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64:
		return t, nil
	}
	return nil, fmt.Errorf("choice requires a scalar type, got %v", t)
}

// Tags reads every occurrence, preserving declaration order and Go escapes.
func Tags(tag reflect.StructTag) ([]string, error) {
	parsed, err := tags.Parse(string(tag))
	if err != nil {
		return nil, err
	}
	var result []string
	for _, item := range parsed {
		if item.Name == "choice" {
			result = append(result, string(item.Values))
		}
	}
	return result, nil
}

// FromStrings converts legacy validation arguments using the declared scalar
// type. Repeated arguments remain valid; membership checkers may deduplicate them.
func FromStrings(t reflect.Type, args []string) (*Constraint, error) {
	typ, err := scalar(t)
	if err != nil {
		return nil, err
	}
	values := make([]interface{}, 0, len(args))
	for _, arg := range args {
		value, err := convert(arg, typ, true)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("choice domain must not be empty")
	}
	return &Constraint{Type: typ, Values: values}, nil
}

func Parse(tag reflect.StructTag, t reflect.Type) (*Constraint, error) {
	// Metadata without choice remains unaffected, even if another tag owner uses
	// a nonstandard spelling. Choice presence invokes the strict tag parser.
	if !strings.Contains(string(tag), `choice:`) {
		return nil, nil
	}
	tags, err := Tags(tag)
	if err != nil {
		return nil, err
	}
	if len(tags) == 0 {
		return nil, nil
	}
	typ, err := scalar(t)
	if err != nil {
		return nil, err
	}
	values := make([]interface{}, 0, len(tags))
	for _, text := range tags {
		v, err := convert(text, typ, true)
		if err != nil {
			return nil, fmt.Errorf("choice %q: %w", text, err)
		}
		values = append(values, v)
	}
	return Normalize(typ, values)
}

// Normalize adapts persisted/schema values to exactly the declared Go scalar.
// It never sorts, deduplicates or coerces numeric-looking strings into numbers.
func Normalize(t reflect.Type, values []interface{}) (*Constraint, error) {
	typ, err := scalar(t)
	if err != nil {
		return nil, err
	}
	c := &Constraint{Type: typ}
	for _, option := range values {
		v, err := convert(option, typ, false)
		if err != nil {
			return nil, err
		}
		for _, old := range c.Values {
			if reflect.DeepEqual(old, v) {
				return nil, fmt.Errorf("duplicate choice %v", v)
			}
		}
		c.Values = append(c.Values, v)
	}
	if len(c.Values) == 0 {
		return nil, fmt.Errorf("choice domain must not be empty")
	}
	return c, nil
}

// Equal includes order: two authors cannot silently replace each other's list.
func Equal(a, b *Constraint) bool {
	return a != nil && b != nil && a.Type == b.Type && reflect.DeepEqual(a.Values, b.Values)
}

func convert(input interface{}, t reflect.Type, textAllowed bool) (interface{}, error) {
	if input == nil {
		return nil, fmt.Errorf("null is absence, not a choice")
	}
	v := reflect.ValueOf(input)
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil, fmt.Errorf("null is absence, not a choice")
		}
		v = v.Elem()
	}
	out := reflect.New(t).Elem()
	kind := t.Kind()
	if kind == reflect.String {
		if v.Kind() != reflect.String || v.Type() == reflect.TypeOf(json.Number("")) {
			return nil, fmt.Errorf("string choice requires string")
		}
		out.SetString(v.String())
		return out.Interface(), nil
	}
	if kind == reflect.Bool {
		if v.Kind() == reflect.Bool {
			out.SetBool(v.Bool())
			return out.Interface(), nil
		}
		if !textAllowed || v.Kind() != reflect.String {
			return nil, fmt.Errorf("boolean choice requires boolean")
		}
		parsed, err := strconv.ParseBool(v.String())
		if err != nil || v.String() != "true" && v.String() != "false" {
			return nil, fmt.Errorf("invalid boolean choice %v", input)
		}
		out.SetBool(parsed)
		return out.Interface(), nil
	}
	var text string
	if n, ok := input.(json.Number); ok {
		text = n.String()
	} else {
		switch v.Kind() {
		case reflect.String:
			if !textAllowed {
				return nil, fmt.Errorf("numeric choice requires numeric value")
			}
			text = v.String()
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			text = strconv.FormatInt(v.Int(), 10)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			text = strconv.FormatUint(v.Uint(), 10)
		case reflect.Float32, reflect.Float64:
			f := v.Float()
			if math.IsNaN(f) || math.IsInf(f, 0) {
				return nil, fmt.Errorf("choice requires finite number")
			}
			if kind != reflect.Float32 && kind != reflect.Float64 && math.Abs(f) > 9007199254740991 {
				return nil, fmt.Errorf("integer choice requires exact integer input, not float64 beyond safe range")
			}
			text = strconv.FormatFloat(f, 'f', -1, v.Type().Bits())
		default:
			return nil, fmt.Errorf("numeric choice requires numeric value")
		}
	}
	switch kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(text, 10, t.Bits())
		if err != nil {
			return nil, fmt.Errorf("invalid %s choice %q: %w", t, text, err)
		}
		out.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(text, 10, t.Bits())
		if err != nil {
			return nil, fmt.Errorf("invalid %s choice %q: %w", t, text, err)
		}
		out.SetUint(n)
	case reflect.Float32, reflect.Float64:
		n, err := strconv.ParseFloat(text, t.Bits())
		if err != nil || math.IsNaN(n) || math.IsInf(n, 0) {
			return nil, fmt.Errorf("invalid finite %s choice %q", t, text)
		}
		out.SetFloat(n)
	default:
		return nil, fmt.Errorf("unsupported choice type %v", t)
	}
	return out.Interface(), nil
}
