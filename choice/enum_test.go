package choice

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestEnumDeclarationsAndIndependentWireStyles(t *testing.T) {
	for _, test := range []struct {
		name          string
		typ           reflect.Type
		tag           reflect.StructTag
		input, output string
		stored        any
	}{
		{"integer storage", reflect.TypeFor[int](), `choice:"0:draft" choice:"10:active" enumAccept:"literal" enumStyle:"ordinal"`, `"active"`, `10`, 10},
		{"string storage", reflect.TypeFor[string](), `choice:"0:draft" choice:"10:active" enumAccept:"ordinal" enumStyle:"literal"`, `10`, `"active"`, "active"},
		{"opposite output", reflect.TypeFor[int](), `choice:"0:draft" choice:"10:active" enumAccept:"ordinal" enumStyle:"literal"`, `10`, `"active"`, 10},
		{"implicit zero", reflect.TypeFor[int](), `choice:"draft" choice:"active" enumAccept:"literal" enumStyle:"ordinal"`, `"draft"`, `0`, 0},
		{"string ordinal output", reflect.TypeFor[string](), `choice:"draft" choice:"active" enumAccept:"literal" enumStyle:"ordinal"`, `"active"`, `1`, "active"},
	} {
		t.Run(test.name, func(t *testing.T) {
			plan, err := ParseEnum(test.tag, test.typ)
			if err != nil || plan == nil {
				t.Fatal(err)
			}
			converted, err := plan.DecodeJSON([]byte(test.input))
			if err != nil {
				t.Fatal(err)
			}
			target := reflect.New(test.typ)
			if err = json.Unmarshal(converted, target.Interface()); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(target.Elem().Interface(), test.stored) {
				t.Fatal(target.Elem().Interface())
			}
			value, err := plan.Encode(target.Elem().Interface())
			if err != nil {
				t.Fatal(err)
			}
			encoded, _ := json.Marshal(value)
			if string(encoded) != test.output {
				t.Fatal(string(encoded))
			}
		})
	}
}
func TestEnumRejectsAmbiguousStyleAndInvalidMappings(t *testing.T) {
	for _, tag := range []reflect.StructTag{
		`choice:"a" enumAccept:"ordinal|literal"`,
		`choice:"a" enumStyle:"literla"`,
		`choice:"0:a" choice:"0:b"`,
		`choice:"0:a" choice:"10:a"`,
		`enumStyle:"literal"`,
	} {
		if _, err := ParseEnum(tag, reflect.TypeFor[int]()); err == nil {
			t.Fatalf("accepted %s", tag)
		}
	}
	plan, err := ParseEnum(`choice:"10:active" enumAccept:"literal"`, reflect.TypeFor[int]())
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{`10`, `"unknown"`, `true`} {
		if _, err := plan.DecodeJSON([]byte(raw)); err == nil {
			t.Fatal(raw)
		}
	}
	numeric, err := Parse(`choice:"3" choice:"7"`, reflect.TypeFor[int]())
	if err != nil || !reflect.DeepEqual(numeric.Values, []any{3, 7}) {
		t.Fatal("legacy numeric choices renumbered", err)
	}
	explicit, err := ParseEnum(`choice:"draft" choice:"10:active" choice:"paused"`, reflect.TypeFor[int]())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(explicit.WireValues(true), []any{int64(0), int64(10), int64(2)}) {
		t.Fatal(explicit.WireValues(true))
	}
}
