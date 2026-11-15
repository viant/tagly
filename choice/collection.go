package choice

import (
	"bytes"
	"encoding"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/viant/tagly/format"
	"github.com/viant/tagly/tags"
)

// ParseCollectionEnum compiles literal choices for an MCP collection.
// It is separate from ParseEnum so ordinary JSON and REST codecs retain their
// existing collection representation and admission rules.
func ParseCollectionEnum(tag reflect.StructTag, t reflect.Type) (*Enum, error) {
	if tag.Get("mcpEnum") != "true" {
		return nil, nil
	}
	base := t
	for base != nil && base.Kind() == reflect.Pointer {
		base = base.Elem()
	}
	if base == nil || base.Kind() != reflect.Slice {
		return nil, nil
	}
	if base.Elem().Kind() != reflect.String {
		switch base.Elem().Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		default:
			return nil, nil
		}
		for _, owner := range []reflect.Type{reflect.TypeOf((*json.Marshaler)(nil)).Elem(), reflect.TypeOf((*json.Unmarshaler)(nil)).Elem(), reflect.TypeOf((*encoding.TextMarshaler)(nil)).Elem(), reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()} {
			if base.Implements(owner) || reflect.PointerTo(base).Implements(owner) {
				return nil, fmt.Errorf("collection enum cannot override custom JSON/Text owner %s", base)
			}
		}
		element, err := parseScalarEnum(tag, base.Elem())
		if err != nil || element == nil {
			return nil, err
		}
		values := element.WireValues(true)
		itemType := reflect.TypeOf("")
		if element.Accept == Ordinal {
			itemType = reflect.TypeOf(int64(0))
		}
		if element.Accept == Boolean {
			itemType = reflect.TypeOf(false)
		}
		return &Enum{Type: base, Accept: element.Accept, Style: element.Style,
			rejectNull: element.rejectNull, collectionElement: element,
			collection: &Constraint{Type: itemType, Values: values}}, nil
	}
	if null, ok := tag.Lookup("enumNull"); ok && null != "reject" {
		return nil, fmt.Errorf("enumNull must be reject, got %q", null)
	}
	scalarTag := tag
	if tag.Get("enumAccept") == "literal" || tag.Get("enumNull") != "" {
		parsed, err := tags.Parse(string(tag))
		if err != nil {
			return nil, err
		}
		retained := parsed[:0]
		for _, item := range parsed {
			if item.Name != "enumAccept" && item.Name != "enumNull" {
				retained = append(retained, item)
			}
		}
		scalarTag = reflect.StructTag(retained.Literal())
	}
	if _, err := parseScalarEnum(scalarTag, t); err != nil {
		return nil, err
	}
	values, err := Tags(tag)
	if err != nil {
		return nil, err
	}
	if len(values) == 0 {
		return nil, nil // The transport policy may have removed the domain.
	}
	policy, err := format.Parse(tag)
	if err != nil {
		return nil, err
	}
	for _, storage := range []reflect.Type{base, base.Elem()} {
		for _, owner := range []reflect.Type{reflect.TypeOf((*json.Marshaler)(nil)).Elem(), reflect.TypeOf((*json.Unmarshaler)(nil)).Elem(), reflect.TypeOf((*encoding.TextMarshaler)(nil)).Elem(), reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()} {
			if storage.Implements(owner) || reflect.PointerTo(storage).Implements(owner) {
				return nil, fmt.Errorf("collection enum cannot override custom JSON/Text owner %s", storage)
			}
		}
	}
	// Collection values are literals, not ordinal mappings. Scalar policy tags
	// continue to be checked by their existing ParseEnum owner.
	constraint, err := FromStrings(base.Elem(), values)
	if err != nil {
		return nil, err
	}
	constraint, err = Normalize(base.Elem(), constraint.Values)
	if err != nil {
		return nil, err
	}
	result := &Enum{Type: base, Accept: Literal, Style: Literal, rejectNull: tag.Get("enumNull") == "reject" || policy.Nullable != nil && !*policy.Nullable, collection: constraint}
	result.compileLiteralMembers(constraint.WireValues())
	return result, nil
}

// CollectionElement returns the compiled scalar item conversion for numeric
// collections, or nil for legacy string collections and scalar enums.
// The descriptor is immutable after compilation, like the containing Enum.
func (e *Enum) CollectionElement() *Enum { return e.collectionElement }

// CollectionItems returns the input item domain, or nil for scalar enums.
func (e *Enum) CollectionItems() *Constraint { return e.collection }

func (e *Enum) decodeCollection(raw []byte) ([]byte, error) {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		if e.rejectNull {
			return nil, ErrNullInput
		}
		return []byte("null"), nil
	}
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, fmt.Errorf("enum collection requires a JSON array: %w", err)
	}
	if e.collectionElement != nil {
		for i, item := range items {
			converted, err := e.collectionElement.DecodeJSON(item)
			if err != nil {
				return nil, fmt.Errorf("enum collection item %d: %w", i, err)
			}
			items[i] = converted
		}
		return json.Marshal(items)
	}
	for i, rawItem := range items {
		if bytes.Equal(bytes.TrimSpace(rawItem), []byte("null")) {
			return nil, fmt.Errorf("enum collection item %d cannot be null", i)
		}
		var value string
		if err := json.Unmarshal(rawItem, &value); err != nil {
			return nil, fmt.Errorf("enum collection item %d requires a JSON string: %w", i, err)
		}
		if !e.literalMembers[value] {
			return nil, fmt.Errorf("unknown enum collection item %d: %q", i, value)
		}
	}
	return raw, nil
}

// encodeCollection delegates each stored element to the compiled scalar owner.
func (e *Enum) encodeCollection(value any) (any, error) {
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
	if v.Kind() != reflect.Slice || v.Type() != e.Type {
		return nil, fmt.Errorf("enum collection requires storage %s", e.Type)
	}
	if v.IsNil() {
		return nil, nil
	}
	result := make([]any, v.Len())
	for i := range result {
		converted, err := e.collectionElement.Encode(v.Index(i).Interface())
		if err != nil {
			return nil, fmt.Errorf("enum collection item %d: %w", i, err)
		}
		result[i] = converted
	}
	return result, nil
}

// compileLiteralMembers prepares exact canonical membership once for the plan.
// Output aliases are deliberately excluded from input membership.
func (e *Enum) compileLiteralMembers(values []any) {
	e.literalMembers = make(map[string]bool, len(values))
	for _, value := range values {
		e.literalMembers[value.(string)] = true
	}
}
