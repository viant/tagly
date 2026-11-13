package choice

import (
	"reflect"
	"testing"
)

func TestHasEnumNestedCollection(t *testing.T) {
	type deal struct {
		Channels []string `json:"channels" mcpEnum:"true" enumAccept:"literal" choice:"DISPLAY" choice:"VIDEO"`
	}
	type input struct {
		Deals []deal `json:"deals"`
	}
	for i := 0; i < 2; i++ {
		if !HasEnum(reflect.TypeFor[input]()) {
			t.Fatal("nested MCP collection must select native enum decoding")
		}
	}
	type legacy struct {
		Channels []string `json:"channels" mcpEnum:"true" choice:"DISPLAY"`
	}
	if HasEnum(reflect.TypeFor[legacy]()) {
		t.Fatal("ordinary collection choices must retain legacy codec")
	}
}

func TestExplicitCollectionEnumUsesOwningCodec(t *testing.T) {
	tag := reflect.StructTag(`mcpEnum:"true" enumAccept:"literal" choice:"DISPLAY" choice:"VIDEO"`)
	enumeration, err := ParseEnum(tag, reflect.TypeFor[[]string]())
	if err != nil || enumeration == nil {
		t.Fatalf("compile: %v", err)
	}
	for _, raw := range []string{`["DISPLAY","VIDEO"]`, `[]`, `null`} {
		converted, err := enumeration.DecodeJSON([]byte(raw))
		if err != nil || string(converted) != raw {
			t.Fatalf("%s: %s %v", raw, converted, err)
		}
	}
	for _, raw := range []string{`["UNKNOWN"]`, `[null]`, `[1]`, `"DISPLAY"`, `{}`} {
		if _, err := enumeration.DecodeJSON([]byte(raw)); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
	if constraint, err := Parse(tag, reflect.TypeFor[[]string]()); err != nil || constraint != nil {
		t.Fatalf("scalar validation must retain collection codec ownership: %v", err)
	}
}
