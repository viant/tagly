package choice

import (
	"reflect"
	"testing"
)

func TestGoJayIntScalarInput(t *testing.T) {
	plan, err := ParseEnum(pacingAliasTag+` enumNumber:"gojayInt"`, reflect.TypeOf((*pacingAlias)(nil)))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ token, want string }{
		{"0.5", `"ASAP"`}, {"1.0", `"SPEND_EVENLY"`}, {"1.5", `"SPEND_EVENLY"`},
		{"1.9", `"SPEND_EVENLY"`}, {"1e0", `"SPEND_EVENLY"`}, {"0e0", `"ASAP"`},
		{"10e-1", `"ASAP"`}, {"1e-1", `"ASAP"`}, {"0.15e1", `"SPEND_EVENLY"`},
		{`"Even"`, `"SPEND_EVENLY"`}, {"null", "null"},
	} {
		got, err := plan.DecodeJSON([]byte(tc.token))
		if err != nil || string(got) != tc.want {
			t.Errorf("%s: got %s (%v), want %s", tc.token, got, err, tc.want)
		}
	}
	for _, token := range []string{"2.1", "1.5e1", "-0", "-0.5", "-0e0", "9223372036854775808", `"1.5"`, "true", "1.5 0"} {
		if _, err := plan.DecodeJSON([]byte(token)); err == nil {
			t.Errorf("accepted %s", token)
		}
	}
	if plan.InputDomain().NumberDescription == "" || plan.WireType(false) != "string" {
		t.Fatal("expected numeric input documentation and literal output")
	}
	plain, err := ParseEnum(pacingAliasTag, reflect.TypeOf(""))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := plain.DecodeJSON([]byte("1.5")); err == nil || plain.InputDomain().NumberDescription != "" {
		t.Fatal("plain scalar enum accepted fractional input or advertised the option")
	}
}

func TestGoJayIntRequiresClosedStringScalar(t *testing.T) {
	for _, tc := range []struct {
		tag reflect.StructTag
		typ reflect.Type
	}{
		{`enumNumber:"gojayInt"`, reflect.TypeOf("")},
		{pacingAliasTag + ` enumNumber:"unknown"`, reflect.TypeOf("")},
		{pacingAliasTag + ` enumNumber:"gojayInt" enumUnknownInteger:"0"`, reflect.TypeOf("")},
		{pacingAliasTag + ` enumNumber:"gojayInt"`, reflect.TypeOf(0)},
		{`choice:"ASAP" mcpEnum:"true" enumNumber:"gojayInt" enumAccept:"literal"`, reflect.TypeOf([]string{})},
	} {
		if _, err := ParseEnum(tc.tag, tc.typ); err == nil {
			t.Errorf("accepted %s on %v", tc.tag, tc.typ)
		}
	}
}
