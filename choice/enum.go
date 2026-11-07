package choice

import (
	"bytes"
	"encoding"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

type Style string

const (
	Ordinal Style = "ordinal"
	Literal Style = "literal"
	// Scalar accepts declared integer ordinals and declared string inputs.
	// It is an opt-in input representation, never an output style.
	Scalar Style = "scalar"
)

type Entry struct {
	Ordinal json.Number
	Literal string
}

// Enum keeps storage independent from input/output JSON representations.
type Enum struct {
	Type               reflect.Type
	Accept, Style      Style
	Entries            []Entry
	aliases            []enumAlias
	fold               bool
	upper              bool
	trimSpace          bool
	integerPassthrough bool
	unknownString      *json.Number
	unknownInteger     *json.Number
}

func enumStyle(raw string, fallback Style) (Style, error) {
	if raw == "" {
		return fallback, nil
	}
	switch Style(raw) {
	case Ordinal, Literal:
		return Style(raw), nil
	}
	return "", fmt.Errorf("enum representation must be ordinal or literal, got %q", raw)
}
func ordinal(text string) (json.Number, error) {
	if value, err := strconv.ParseInt(text, 10, 64); err == nil {
		return json.Number(strconv.FormatInt(value, 10)), nil
	}
	if value, err := strconv.ParseUint(text, 10, 64); err == nil {
		return json.Number(strconv.FormatUint(value, 10)), nil
	}
	return "", fmt.Errorf("invalid enum ordinal %q", text)
}
func defaultStyle(t reflect.Type) Style {
	if t.Kind() == reflect.String {
		return Literal
	}
	return Ordinal
}
func ParseEnum(tag reflect.StructTag, t reflect.Type) (*Enum, error) {
	if !strings.Contains(string(tag), `choice:`) && !strings.Contains(string(tag), `enumStyle:`) && !strings.Contains(string(tag), `enumAccept:`) && !strings.Contains(string(tag), `enumAlias:`) && !strings.Contains(string(tag), `enumCase:`) && !strings.Contains(string(tag), `enumInteger:`) && !strings.Contains(string(tag), `enumUnknownString:`) && !strings.Contains(string(tag), `enumUnknownInteger:`) && !strings.Contains(string(tag), `enumTrim:`) {
		return nil, nil
	}
	values, err := Tags(tag)
	if err != nil {
		return nil, err
	}
	_, hasStyle := tag.Lookup("enumStyle")
	_, hasAccept := tag.Lookup("enumAccept")
	_, hasAlias := tag.Lookup("enumAlias")
	_, hasCase := tag.Lookup("enumCase")
	_, hasTrim := tag.Lookup("enumTrim")
	base := t
	for base != nil && base.Kind() == reflect.Pointer {
		base = base.Elem()
	}
	_, hasInteger := tag.Lookup("enumInteger")
	_, hasUnknown := tag.Lookup("enumUnknownString")
	_, hasUnknownInteger := tag.Lookup("enumUnknownInteger")
	active := hasStyle || hasAccept || hasAlias || hasCase || hasTrim || hasInteger || hasUnknown || hasUnknownInteger
	legacyNumeric := base != nil && base.Kind() != reflect.String
	for _, value := range values {
		if prefix, _, ok := strings.Cut(value, ":"); ok {
			if _, e := ordinal(prefix); e == nil {
				active = true
				legacyNumeric = false
			}
		}
		if _, e := ordinal(value); e != nil {
			if _, numeric := strconv.ParseFloat(value, 64); numeric != nil {
				legacyNumeric = false
			}
		}
	}
	if !active && base != nil && base.Kind() != reflect.String && len(values) > 0 && !legacyNumeric {
		switch base.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			active = true
		}
	}
	if !active {
		return nil, nil
	}
	if base == nil {
		return nil, fmt.Errorf("enum requires a storage type")
	}
	switch base.Kind() {
	case reflect.String, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
	default:
		return nil, fmt.Errorf("enum requires integer or string storage, got %s", base)
	}
	for _, owner := range []reflect.Type{reflect.TypeOf((*json.Marshaler)(nil)).Elem(), reflect.TypeOf((*json.Unmarshaler)(nil)).Elem(), reflect.TypeOf((*encoding.TextMarshaler)(nil)).Elem(), reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()} {
		if base.Implements(owner) || reflect.PointerTo(base).Implements(owner) {
			return nil, fmt.Errorf("enum cannot override custom JSON/Text owner %s", base)
		}
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("enum representation requires choice declarations")
	}
	result := &Enum{Type: base}
	if tag.Get("enumAccept") == string(Scalar) {
		result.Accept = Scalar
	} else {
		result.Accept, err = enumStyle(tag.Get("enumAccept"), defaultStyle(base))
	}
	if err != nil {
		return nil, err
	}
	result.Style, err = enumStyle(tag.Get("enumStyle"), defaultStyle(base))
	if err != nil {
		return nil, err
	}
	seenOrd, seenLiteral := map[string]bool{}, map[string]bool{}
	for index, value := range values {
		code := json.Number(strconv.Itoa(index))
		label := value
		if prefix, literal, ok := strings.Cut(value, ":"); ok {
			if explicit, e := ordinal(prefix); e == nil {
				code, label = explicit, literal
			}
		} else if legacyNumeric {
			code, err = ordinal(value)
			if err != nil {
				return nil, err
			}
		}
		if seenOrd[string(code)] || seenLiteral[label] {
			return nil, fmt.Errorf("duplicate enum ordinal or literal: %s:%s", code, label)
		}
		seenOrd[string(code)], seenLiteral[label] = true, true
		result.Entries = append(result.Entries, Entry{code, label})
		if base.Kind() != reflect.String {
			if _, err = result.storage(code); err != nil {
				return nil, err
			}
		}
	}
	if err := result.parseAliases(tag); err != nil {
		return nil, err
	}
	return result, nil
}
func (e *Enum) storage(value any) (any, error) {
	if e.Type.Kind() == reflect.String {
		v := reflect.New(e.Type).Elem()
		v.SetString(value.(string))
		return v.Interface(), nil
	}
	return convert(value, e.Type, true)
}
func (e *Enum) StorageValues() ([]any, error) {
	values := make([]any, 0, len(e.Entries))
	for _, entry := range e.Entries {
		var value any = entry.Ordinal
		if e.Type.Kind() == reflect.String {
			value = entry.Literal
		}
		stored, err := e.storage(value)
		if err != nil {
			return nil, err
		}
		values = append(values, stored)
	}
	return values, nil
}
func (e *Enum) WireType(input bool) string {
	style := e.Style
	if input {
		style = e.Accept
	}
	if style == Scalar {
		// A scalar union has no single JSON type. InputDomain describes its
		// branches; callers must not advertise an integer-only contract.
		return ""
	}
	if style == Literal {
		return "string"
	}
	return "integer"
}
func (e *Enum) WireValues(input bool) []any {
	if !input && e.Style == Ordinal {
		if e.integerPassthrough {
			return nil
		}
		if e.unknownString != nil {
			// Preserve existing open ordinal policies. An integer fallback can
			// advertise finite output when every fallback is a declared choice.
			if e.unknownInteger == nil {
				return nil
			}
			if _, err := e.declaredFallback(string(*e.unknownString)); err != nil {
				return nil
			}
		}
	}
	style := e.Style
	if input {
		style = e.Accept
	}
	if style == Scalar {
		// Folded string domains cannot be represented by a finite JSON enum.
		return nil
	}
	result := make([]any, 0, len(e.Entries))
	for _, entry := range e.Entries {
		if style == Literal {
			result = append(result, entry.Literal)
		} else if n, err := strconv.ParseInt(string(entry.Ordinal), 10, 64); err == nil {
			result = append(result, n)
		} else {
			n, _ := strconv.ParseUint(string(entry.Ordinal), 10, 64)
			result = append(result, n)
		}
	}
	return result
}
func (e *Enum) Encode(value any) (any, error) {
	v := reflect.ValueOf(value)
	for v.IsValid() && v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil, nil
		}
		v = v.Elem()
	}
	if !v.IsValid() {
		return nil, nil
	}
	if e.Type.Kind() == reflect.String {
		literal := v.String()
		if e.Style == Literal {
			return literal, nil
		}
		for _, entry := range e.Entries {
			if entry.Literal == literal {
				return entry.Ordinal, nil
			}
		}
		return nil, fmt.Errorf("unknown enum literal %q", literal)
	}
	var text string
	if v.Kind() >= reflect.Int && v.Kind() <= reflect.Int64 {
		text = strconv.FormatInt(v.Int(), 10)
	} else {
		text = strconv.FormatUint(v.Uint(), 10)
	}
	if e.Style == Ordinal {
		return json.Number(text), nil
	}
	for _, entry := range e.Entries {
		if string(entry.Ordinal) == text {
			return entry.Literal, nil
		}
	}
	return nil, fmt.Errorf("unknown enum ordinal %s", text)
}

// DecodeJSON returns JSON in the canonical storage representation. Null and
// absence remain owned by the selected decoder, not by enumeration metadata.
func (e *Enum) DecodeJSON(raw []byte) ([]byte, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("unexpected trailing enum JSON value")
	}
	if value == nil {
		return []byte("null"), nil
	}
	if e.Accept == Scalar {
		return e.decodeScalar(value)
	}
	var stored any
	if e.Accept == Literal {
		literal, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("enum input requires a JSON string")
		}
		if e.Type.Kind() == reflect.String {
			stored = literal
		} else {
			for _, entry := range e.Entries {
				if entry.Literal == literal {
					stored = entry.Ordinal
					break
				}
			}
		}
	} else {
		number, ok := value.(json.Number)
		if !ok {
			return nil, fmt.Errorf("enum input requires a JSON integer")
		}
		code, err := ordinal(string(number))
		if err != nil {
			return nil, err
		}
		if e.Type.Kind() != reflect.String {
			stored = code
		} else {
			for _, entry := range e.Entries {
				if entry.Ordinal == code {
					stored = entry.Literal
					break
				}
			}
		}
	}
	if stored == nil {
		return nil, fmt.Errorf("unknown enum input %v", value)
	}
	if _, err := e.storage(stored); err != nil {
		return nil, err
	}
	return json.Marshal(stored)
}

var enumTypes sync.Map

// HasEnum reports whether linked JSON fields need enum conversion. Unchanged
// types use a cached scan; custom JSON owners keep their own complete contract.
func HasEnum(t reflect.Type) bool {
	if t == nil {
		return false
	}
	if v, ok := enumTypes.Load(t); ok {
		return v.(bool)
	}
	seen := map[reflect.Type]bool{}
	var visit func(reflect.Type) bool
	visit = func(t reflect.Type) bool {
		if seen[t] {
			return false
		}
		seen[t] = true
		switch t.Kind() {
		case reflect.Pointer, reflect.Slice, reflect.Array, reflect.Map:
			return visit(t.Elem())
		}
		if t.Kind() != reflect.Struct {
			return false
		}
		for _, owner := range []reflect.Type{reflect.TypeOf((*json.Marshaler)(nil)).Elem(), reflect.TypeOf((*json.Unmarshaler)(nil)).Elem()} {
			if t.Implements(owner) || reflect.PointerTo(t).Implements(owner) {
				return false
			}
		}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if (!f.IsExported() && !f.Anonymous) || f.Tag.Get("json") == "-" || f.Tag.Get("setMarker") == "true" {
				continue
			}
			if e, err := ParseEnum(f.Tag, f.Type); e != nil || err != nil {
				return true
			}
			if visit(f.Type) {
				return true
			}
		}
		return false
	}
	result := visit(t)
	enumTypes.Store(t, result)
	return result
}
