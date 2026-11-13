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

// ParseCollectionEnum compiles literal choices for an MCP string collection.
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
	if base == nil || base.Kind() != reflect.Slice || base.Elem().Kind() != reflect.String {
		return nil, nil
	}
	scalarTag := tag
	if tag.Get("enumAccept") == "literal" {
		parsed, err := tags.Parse(string(tag))
		if err != nil {
			return nil, err
		}
		retained := parsed[:0]
		for _, item := range parsed {
			if item.Name != "enumAccept" {
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
	result := &Enum{Type: base, Accept: Literal, Style: Literal, rejectNull: policy.Nullable != nil && !*policy.Nullable, collection: constraint, collectionMembers: make(map[string]bool, len(values))}
	for _, value := range constraint.WireValues() {
		result.collectionMembers[value.(string)] = true
	}
	return result, nil
}

// CollectionItems returns the literal item domain, or nil for scalar enums.
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
	for i, rawItem := range items {
		if bytes.Equal(bytes.TrimSpace(rawItem), []byte("null")) {
			return nil, fmt.Errorf("enum collection item %d cannot be null", i)
		}
		var value string
		if err := json.Unmarshal(rawItem, &value); err != nil {
			return nil, fmt.Errorf("enum collection item %d requires a JSON string: %w", i, err)
		}
		if !e.collectionMembers[value] {
			return nil, fmt.Errorf("unknown enum collection item %d: %q", i, value)
		}
	}
	return raw, nil
}
