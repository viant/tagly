package choice

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/viant/tagly/tags"
)

type scalarUnionPolicy struct {
	integers, strings, zeroNull, emptyNull bool
	pointer                                bool
}

func parseScalarUnion(tag reflect.StructTag, typ reflect.Type, choices []string, pointer bool) (*Enum, error) {
	if tag.Get("enumAccept") != "scalar" || tag.Get("enumStyle") != "scalar" {
		return nil, fmt.Errorf("ScalarValue requires scalar input and output styles")
	}
	p := &scalarUnionPolicy{pointer: pointer}
	e := &Enum{Type: typ, Accept: Scalar, Style: Scalar, union: p}
	seen := map[int64]bool{}
	for _, choice := range choices {
		code, literal, ok := strings.Cut(choice, ":")
		if !ok {
			return nil, fmt.Errorf("scalar union choice requires integer:literal")
		}
		id, err := strconv.ParseInt(code, 10, 64)
		if err != nil {
			return nil, err
		}
		if seen[id] {
			return nil, fmt.Errorf("duplicate scalar union ordinal %d", id)
		}
		seen[id] = true
		e.Entries = append(e.Entries, Entry{Ordinal: json.Number(strconv.FormatInt(id, 10)), Literal: literal})
	}
	parsed, err := tags.Parse(string(tag))
	if err != nil {
		return nil, err
	}
	// Reuse the existing alias/case/trim compiler. Union-specific policies
	// retain the source branch instead of applying enum fallback normalization.
	var aliasTags strings.Builder
	for _, item := range parsed {
		value := string(item.Values)
		switch item.Name {
		case "enumAlias", "enumCase", "enumTrim":
			fmt.Fprintf(&aliasTags, "%s:%s ", item.Name, strconv.Quote(value))
		case "enumInteger":
			if value != "passthrough" {
				return nil, fmt.Errorf("unsupported union enumInteger %q", value)
			}
			p.integers = true
		case "enumUnknownString":
			if value != "preserve" {
				return nil, fmt.Errorf("unsupported union enumUnknownString %q", value)
			}
			p.strings = true
		case "enumZero":
			if value != "null" {
				return nil, fmt.Errorf("unsupported enumZero %q", value)
			}
			p.zeroNull = true
		case "enumEmpty":
			if value != "null" {
				return nil, fmt.Errorf("unsupported enumEmpty %q", value)
			}
			p.emptyNull = true
		case "enumUnknownInteger":
			return nil, fmt.Errorf("scalar union cannot apply integer fallback")
		}
	}
	if len(choices) == 0 && !p.integers && !p.strings {
		return nil, fmt.Errorf("scalar union requires choices or an explicit passthrough policy")
	}
	if err := e.parseAliases(reflect.StructTag(aliasTags.String())); err != nil {
		return nil, err
	}
	return e, nil
}

func (e *Enum) normalizeScalarUnion(value ScalarValue) (ScalarValue, error) {
	if _, err := value.Value(); err != nil {
		return ScalarValue{}, err
	}
	if value.Integer != nil {
		if *value.Integer == 0 && e.union.zeroNull {
			return ScalarValue{}, nil
		}
		if e.union.integers {
			return value, nil
		}
		for _, entry := range e.Entries {
			if string(entry.Ordinal) == strconv.FormatInt(*value.Integer, 10) {
				return value, nil
			}
		}
		return ScalarValue{}, fmt.Errorf("unknown scalar union integer %d", *value.Integer)
	}
	if value.String != nil {
		text := *value.String
		if e.trimSpace {
			text = strings.TrimSpace(text)
		}
		if text == "" && e.union.emptyNull {
			return ScalarValue{}, nil
		}
		for _, alias := range e.aliases {
			if e.equalInput(alias.text, text) {
				canonical := e.Entries[alias.entry].Literal
				return ScalarValue{String: &canonical}, nil
			}
		}
		if !e.union.strings {
			return ScalarValue{}, fmt.Errorf("unknown scalar union string %q", text)
		}
		return ScalarValue{String: &text}, nil
	}
	return value, nil
}

func (e *Enum) decodeScalarUnion(raw []byte) ([]byte, error) {
	value, err := ParseScalarValue(raw)
	if err != nil {
		return nil, err
	}
	if e.rejectNull && value.String == nil && value.Integer == nil {
		return nil, ErrNullInput
	}
	value, err = e.normalizeScalarUnion(value)
	if err != nil {
		return nil, err
	}
	if value.String == nil && value.Integer == nil && e.union.pointer {
		return []byte("null"), nil
	}
	return json.Marshal(value)
}

func (e *Enum) encodeScalarUnion(value ScalarValue) (any, error) {
	value, err := e.normalizeScalarUnion(value)
	if err != nil {
		return nil, err
	}
	if value.Integer != nil {
		for _, entry := range e.Entries {
			if string(entry.Ordinal) == strconv.FormatInt(*value.Integer, 10) {
				return entry.Literal, nil
			}
		}
	}
	return value.Value()
}

func (e *Enum) scalarUnionDomain() *InputDomain {
	d := &InputDomain{OpenIntegers: e.union.integers, OpenStrings: e.union.strings,
		Nullable: true, Minimum: "-9223372036854775808", Maximum: "9223372036854775807"}
	for _, entry := range e.Entries {
		d.Integers = append(d.Integers, entry.Ordinal)
	}
	if e.union.zeroNull {
		found := false
		for _, id := range d.Integers {
			if id == "0" {
				found = true
			}
		}
		if !found {
			d.Integers = append(d.Integers, "0")
		}
	}
	for _, alias := range e.aliases {
		d.Strings = append(d.Strings, alias.text)
		pattern := enumInputPattern(alias.text, e.fold, e.upper)
		if e.trimSpace {
			pattern = trimSpacePattern + "*(?:" + pattern + ")" + trimSpacePattern + "*"
		}
		d.StringAlternatives = append(d.StringAlternatives, StringAlternative{Pattern: "^(?:" + pattern + ")$", Length: len([]rune(alias.text)), VariableLength: e.trimSpace})
	}
	if e.union.emptyNull {
		d.Strings = append(d.Strings, "")
		pattern := "^(?:)$"
		if e.trimSpace {
			pattern = "^(?:" + trimSpacePattern + "*)$"
		}
		d.StringAlternatives = append(d.StringAlternatives, StringAlternative{Pattern: pattern, Length: 0, VariableLength: e.trimSpace})
	}
	return d
}

// OutputDomain describes union outputs. Nil means the existing WireType and
// WireValues APIs describe the output. Consumers must handle this before
// advertising a single JSON type for branch-preserving scalar storage.
func (e *Enum) OutputDomain() *InputDomain {
	if e.union == nil {
		return nil
	}
	d := e.scalarUnionDomain()
	// Declared integers are emitted as canonical strings, not finite integer
	// alternatives. With passthrough, the bounded integer branch remains open.
	d.Integers = nil
	d.Strings, d.StringAlternatives = nil, nil
	seen := map[string]bool{}
	for _, entry := range e.Entries {
		if !seen[entry.Literal] {
			d.Strings = append(d.Strings, entry.Literal)
			seen[entry.Literal] = true
		}
	}
	return d
}
