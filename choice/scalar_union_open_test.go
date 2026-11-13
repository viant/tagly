package choice

import (
	"encoding/json"
	"reflect"
	"testing"
)

const openUnionTag reflect.StructTag = `enumAccept:"scalar" enumStyle:"scalar" enumInteger:"passthrough" enumUnknownString:"preserve"`

func TestScalarUnionWithoutChoicesUsesExistingOpenPolicies(t *testing.T) {
	for _, typ := range []reflect.Type{reflect.TypeFor[ScalarValue](), reflect.TypeFor[*ScalarValue]()} {
		plan, err := ParseEnum(openUnionTag, typ)
		if err != nil {
			t.Fatal(err)
		}
		for _, domain := range []*InputDomain{plan.InputDomain(), plan.OutputDomain()} {
			if domain == nil || !domain.OpenIntegers || !domain.OpenStrings || !domain.Nullable || domain.Minimum != "-9223372036854775808" || domain.Maximum != "9223372036854775807" {
				t.Fatalf("open domain = %+v", domain)
			}
		}
		if len(plan.Entries) != 0 || len(plan.WireValues(false)) != 0 {
			t.Fatal("an open scalar must not invent a finite choice")
		}
		for _, raw := range []string{`0`, `-7`, `9223372036854775807`, `-9223372036854775808`, `""`, `"plain text"`, `"1e2"`, `" 50 "`, `null`} {
			decoded, err := plan.DecodeJSON([]byte(raw))
			if err != nil {
				t.Fatalf("%v %s: %v", typ, raw, err)
			}
			var stored ScalarValue
			if err := json.Unmarshal(decoded, &stored); err != nil {
				t.Fatal(err)
			}
			value, err := plan.Encode(stored)
			if err != nil {
				t.Fatal(err)
			}
			encoded, err := json.Marshal(value)
			if err != nil || string(encoded) != raw {
				t.Fatalf("%s -> %s, %v", raw, encoded, err)
			}
		}
		for _, raw := range []string{`true`, `{}`, `[]`, `1.5`, `1e2`, `9223372036854775808`, `-9223372036854775809`, `1 2`} {
			if _, err := plan.DecodeJSON([]byte(raw)); err == nil {
				t.Fatalf("accepted non-integer/string input %s", raw)
			}
		}
	}
}

func TestScalarUnionWithoutChoicesRetainsIndependentPolicies(t *testing.T) {
	for _, tc := range []struct {
		tag       reflect.StructTag
		good, bad string
	}{
		{`enumAccept:"scalar" enumStyle:"scalar" enumInteger:"passthrough"`, `99`, `"99"`},
		{`enumAccept:"scalar" enumStyle:"scalar" enumUnknownString:"preserve"`, `"99"`, `99`},
	} {
		plan, err := ParseEnum(tc.tag, reflect.TypeFor[ScalarValue]())
		if err != nil {
			t.Fatal(err)
		}
		if _, err := plan.DecodeJSON([]byte(tc.good)); err != nil {
			t.Fatal(err)
		}
		if _, err := plan.DecodeJSON([]byte(tc.bad)); err == nil {
			t.Fatalf("accepted disabled scalar branch %s", tc.bad)
		}
	}
	for _, tag := range []reflect.StructTag{
		`enumAccept:"scalar" enumStyle:"scalar"`,
		`enumAccept:"scalar" enumStyle:"scalar" enumInteger:"loose"`,
		`enumAccept:"scalar" enumStyle:"scalar" enumInteger:"passthrough" enumAlias:"1:one"`,
	} {
		if _, err := ParseEnum(tag, reflect.TypeFor[ScalarValue]()); err == nil {
			t.Fatalf("accepted invalid open declaration %s", tag)
		}
	}
}
