package choice

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestMCPCollectionLiteralDomain(t *testing.T) {
	tag := reflect.StructTag(`mcpEnum:"true" choice:"DISPLAY" choice:"VIDEO"`)
	for _, typ := range []reflect.Type{reflect.TypeOf([]string{}), reflect.TypeOf((*[]string)(nil))} {
		plan, err := ParseCollectionEnum(tag, typ)
		if err != nil || plan == nil {
			t.Fatalf("plan %v: %v", typ, err)
		}
		if !reflect.DeepEqual(plan.CollectionItems().WireValues(), []any{"DISPLAY", "VIDEO"}) {
			t.Fatal(plan.CollectionItems())
		}
		if plan.WireType(true) != "array" || plan.WireValues(true) != nil {
			t.Fatal("collection domain must belong to items")
		}
		if value, err := plan.Encode([]string{"DISPLAY"}); err != nil || !reflect.DeepEqual(value, []string{"DISPLAY"}) {
			t.Fatal(value, err)
		}
		if _, err := plan.Encode([]string{"UNKNOWN"}); err == nil {
			t.Fatal("unknown encoded collection item")
		}
		for _, raw := range []string{`["DISPLAY","VIDEO"]`, `[]`, `null`} {
			got, err := plan.DecodeJSON([]byte(raw))
			if err != nil || string(got) != raw {
				t.Fatalf("%s: %s %v", raw, got, err)
			}
		}
		for _, raw := range []string{`["UNKNOWN"]`, `[null]`, `[1]`, `"DISPLAY"`, `{}`, `["DISPLAY"] []`} {
			if _, err := plan.DecodeJSON([]byte(raw)); err == nil {
				t.Errorf("admitted %s", raw)
			}
		}
		if scalar, err := ParseEnum(tag, typ); err != nil || scalar != nil {
			t.Fatalf("REST enum changed: %v %v", scalar, err)
		}
		if constraint, err := Parse(tag, typ); err != nil || constraint != nil {
			t.Fatalf("REST validation changed: %v %v", constraint, err)
		}
	}
	var ordinary struct {
		Values []string `json:"values" mcpEnum:"true" choice:"DISPLAY"`
	}
	if err := json.Unmarshal([]byte(`{"values":["UNKNOWN"]}`), &ordinary); err != nil || ordinary.Values[0] != "UNKNOWN" {
		t.Fatal(ordinary, err)
	}
}

func TestMCPCollectionGateAndScalarCompatibility(t *testing.T) {
	strict, err := ParseCollectionEnum(`mcpEnum:"true" choice:"DISPLAY" format:"nullable=false"`, reflect.TypeOf([]string{}))
	if err != nil || strict == nil || strict.AllowsNullInput() {
		t.Fatal(strict, err)
	}
	if _, err := strict.DecodeJSON([]byte("null")); !errors.Is(err, ErrNullInput) {
		t.Fatal(err)
	}
	if e, err := ParseCollectionEnum(`mcpEnum:"true"`, reflect.TypeOf([]string{})); err != nil || e != nil {
		t.Fatal("stripped REST domain", e, err)
	}
	if e, err := ParseCollectionEnum(`choice:"DISPLAY"`, reflect.TypeOf([]string{})); err != nil || e != nil {
		t.Fatal(e, err)
	}
	if _, err := Parse(`choice:"DISPLAY"`, reflect.TypeOf([]string{})); err == nil {
		t.Fatal("unmarked collection behavior changed")
	}
	if _, err := ParseCollectionEnum(`mcpEnum:"true" choice:"DISPLAY" choice:"DISPLAY"`, reflect.TypeOf([]string{})); err == nil {
		t.Fatal("duplicate items")
	}
	tag := reflect.StructTag(`mcpEnum:"true" choice:"1:ACTIVE" enumAccept:"scalar" enumStyle:"literal"`)
	if e, err := ParseCollectionEnum(tag, reflect.TypeOf("")); err != nil || e != nil {
		t.Fatal(e, err)
	}
	if e, err := ParseEnum(tag, reflect.TypeOf("")); err != nil || e == nil {
		t.Fatal(e, err)
	}
	if _, err := Parse(`mcpEnum:"true" choice:"DISPLAY" enumAccept:"literal"`, reflect.TypeOf([]string{})); err == nil || !strings.Contains(err.Error(), "storage") {
		t.Fatal(err)
	}
}
