package choice

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

func TestEnumNullRejectionIsOptIn(t *testing.T) {
	for _, typ := range []reflect.Type{reflect.TypeFor[int](), reflect.TypeFor[*int](), reflect.TypeFor[string]()} {
		for _, reject := range []bool{false, true} {
			tag := reflect.StructTag(`choice:"0:ASAP" choice:"1:SPEND_EVENLY" enumAccept:"literal" enumStyle:"literal"`)
			if reject {
				tag += ` enumNull:"reject"`
			}
			plan, err := ParseEnum(tag, typ)
			if err != nil {
				t.Fatal(err)
			}
			if plan.AllowsNullInput() == reject {
				t.Fatal("null admission does not match the explicit policy")
			}
			value, err := plan.DecodeJSON([]byte(" \n null \t"))
			if reject {
				if !errors.Is(err, ErrNullInput) {
					t.Fatalf("%v: got %v", typ, err)
				}
			} else if err != nil || string(value) != "null" {
				t.Fatalf("default null handling changed: %s %v", value, err)
			}
			value, err = plan.DecodeJSON([]byte(`"ASAP"`))
			if err != nil {
				t.Fatal(err)
			}
			expected := "0"
			if typ.Kind() == reflect.String {
				expected = `"ASAP"`
			}
			if string(value) != expected {
				t.Fatalf("explicit zero changed: %s", value)
			}
			var stored any = 0
			if typ.Kind() == reflect.String {
				stored = "ASAP"
			}
			encoded, err := plan.Encode(stored)
			if err != nil || encoded != "ASAP" {
				t.Fatalf("output changed: %v %v", encoded, err)
			}
			if _, err = plan.DecodeJSON([]byte(`null true`)); err == nil {
				t.Fatal("trailing JSON must still be rejected")
			}
		}
	}
}

func TestEnumNullScalarUnionPolicy(t *testing.T) {
	for _, typ := range []reflect.Type{reflect.TypeFor[ScalarValue](), reflect.TypeFor[*ScalarValue]()} {
		plan, err := ParseEnum(`choice:"1:ONE" enumAccept:"scalar" enumStyle:"scalar" enumNull:"reject" enumEmpty:"null" enumZero:"null"`, typ)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = plan.DecodeJSON([]byte(`null`)); !errors.Is(err, ErrNullInput) {
			t.Fatal(err)
		}
		if plan.InputDomain().Nullable {
			t.Fatal("input domain permits explicit null")
		}
		if !plan.OutputDomain().Nullable {
			t.Fatal("input rule changed the stored output domain")
		}
		// Explicit empty/zero conversion policies remain independent of JSON null.
		for _, raw := range []string{`""`, `0`, `"ONE"`, `1`} {
			converted, err := plan.DecodeJSON([]byte(raw))
			if err != nil {
				t.Fatalf("%s: %v", raw, err)
			}
			if raw == `""` || raw == `0` {
				if typ.Kind() == reflect.Pointer {
					if string(converted) != "null" {
						t.Fatalf("%s: expected normalized null, got %s", raw, converted)
					}
				} else {
					var stored ScalarValue
					if err = json.Unmarshal(converted, &stored); err != nil {
						t.Fatal(err)
					}
					if stored.Integer != nil || stored.String != nil {
						t.Fatalf("%s: expected null storage, got %+v", raw, stored)
					}
				}
			}
		}
	}
}

func TestEnumNullDeclarationValidation(t *testing.T) {
	for _, tag := range []reflect.StructTag{
		`choice:"ZERO" enumNull:""`,
		`choice:"ZERO" enumNull:"ignore"`,
		`enumNull:"reject"`,
	} {
		if _, err := ParseEnum(tag, reflect.TypeFor[int]()); err == nil {
			t.Fatalf("accepted invalid declaration %s", tag)
		}
	}
}
