package choice

import (
	"reflect"
	"regexp"
	"testing"
)

const openScalarTag reflect.StructTag = `choice:"1:FIXED_BUDGET" choice:"2:AUTO_BUDGET" enumAccept:"scalar" enumStyle:"ordinal" enumInteger:"passthrough" enumUnknownString:"0" enumCase:"upper"`

func TestOpenScalarCompatibility(t *testing.T) {
	plan, err := ParseEnum(openScalarTag, reflect.TypeOf(int64(0)))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ raw, want string }{{`0`, `0`}, {`-0`, `0`}, {`3`, `3`}, {`-1`, `-1`}, {`9223372036854775807`, `9223372036854775807`}, {`-9223372036854775808`, `-9223372036854775808`}, {`"FIXED_BUDGET"`, `1`}, {`"fixed_budget"`, `1`}, {`"fıxed_budget"`, `1`}, {`"AUTO_BUDGET"`, `2`}, {`""`, `0`}, {`"other"`, `0`}, {`"1"`, `0`}, {`" FIXED_BUDGET"`, `0`}, {`null`, `null`}} {
		got, err := plan.DecodeJSON([]byte(tc.raw))
		if err != nil || string(got) != tc.want {
			t.Errorf("%s -> %s err=%v want%s", tc.raw, got, err, tc.want)
		}
	}
	for _, raw := range []string{`1.0`, `1.5`, `1e0`, `true`, `[]`, `{}`, `9223372036854775808`, `-9223372036854775809`, `1 2`} {
		if got, err := plan.DecodeJSON([]byte(raw)); err == nil {
			t.Errorf("accepted %s as%s", raw, got)
		}
	}
	if values := plan.WireValues(false); len(values) != 0 {
		t.Errorf("open ordinal output has finite enum%v", values)
	}
}
func TestOpenScalarStorageWidthAndPolicies(t *testing.T) {
	plan, err := ParseEnum(openScalarTag, reflect.TypeOf(int8(0)))
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{`128`, `-129`} {
		if _, err := plan.DecodeJSON([]byte(raw)); err == nil {
			t.Errorf("accepted %s forint8", raw)
		}
	}
	for _, tag := range []reflect.StructTag{`enumInteger:"passthrough"`, `choice:"1:A" enumAccept:"ordinal" enumInteger:"passthrough"`, `choice:"1:A" enumAccept:"scalar" enumInteger:"loose"`, `choice:"1:A" enumAccept:"scalar" enumUnknownString:"128"`, `choice:"1:A" enumAccept:"scalar" enumUnknownString:"1.0"`, `choice:"1:A" enumAccept:"scalar" enumStyle:"literal" enumInteger:"passthrough"`, `choice:"1:I" choice:"2:ı" enumAccept:"scalar" enumUnknownString:"0" enumCase:"upper"`} {
		if _, err := ParseEnum(tag, reflect.TypeOf(int8(0))); err == nil {
			t.Errorf("accepted bad declaration %s", tag)
		}
	}
	if _, err := ParseEnum(openScalarTag, reflect.TypeOf("")); err == nil {
		t.Error("accepted integer passthrough forstring storage")
	}
}

func TestOpenScalarDomainAndIndependentPolicies(t *testing.T) {
	plan, err := ParseEnum(openScalarTag, reflect.TypeOf(int8(0)))
	if err != nil {
		t.Fatal(err)
	}
	domain := plan.InputDomain()
	if !domain.OpenIntegers || !domain.OpenStrings || domain.Minimum != "-128" || domain.Maximum != "127" || len(domain.StringAlternatives) != 0 {
		t.Fatalf("domain=%+v", domain)
	}
	for _, tc := range []struct {
		tag       reflect.StructTag
		good, bad string
	}{
		{`choice:"1:A" enumAccept:"scalar" enumInteger:"passthrough"`, `99`, `"other"`},
		{`choice:"1:A" enumAccept:"scalar" enumUnknownString:"0"`, `"other"`, `99`},
	} {
		p, err := ParseEnum(tc.tag, reflect.TypeOf(int(0)))
		if err != nil {
			t.Fatal(err)
		}
		if _, err = p.DecodeJSON([]byte(tc.good)); err != nil {
			t.Fatal(err)
		}
		if _, err = p.DecodeJSON([]byte(tc.bad)); err == nil {
			t.Fatalf("accepted %s", tc.bad)
		}
	}
	upper, err := ParseEnum(`choice:"1:I" enumAccept:"scalar" enumCase:"upper"`, reflect.TypeOf(int(0)))
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{`"I"`, `"i"`, `"ı"`} {
		if got, err := upper.DecodeJSON([]byte(raw)); err != nil || string(got) != "1" {
			t.Fatalf("%s %s %v", raw, got, err)
		}
	}
	pattern := upper.InputDomain().StringAlternatives[0].Pattern
	re, err := regexp.Compile(pattern)
	if err != nil {
		t.Fatal(err)
	}
	if !re.MatchString("ı") || re.MatchString("İ") {
		t.Fatal(pattern)
	}
}
