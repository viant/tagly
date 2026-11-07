package choice

import (
	"reflect"
	"strconv"
	"testing"
)

const stringFallbackTag reflect.StructTag = `choice:"0:ASAP" choice:"1:SPEND_EVENLY" enumAccept:"scalar" enumStyle:"literal" enumCase:"fold" enumUnknownString:"0" enumUnknownInteger:"0"`

func TestScalarStringFallback(t *testing.T) {
	plan, err := ParseEnum(stringFallbackTag, reflect.TypeOf(""))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range [][2]string{{`0`, `"ASAP"`}, {`-0`, `"ASAP"`}, {`1`, `"SPEND_EVENLY"`}, {`2`, `"ASAP"`}, {`-1`, `"ASAP"`}, {`"sPeNd_EvEnLy"`, `"SPEND_EVENLY"`}, {`"ſPEND_EVENLY"`, `"SPEND_EVENLY"`}, {`"Even"`, `"ASAP"`}, {`"1"`, `"ASAP"`}, {`""`, `"ASAP"`}, {`" SPEND_EVENLY"`, `"ASAP"`}, {`null`, `null`}} {
		got, err := plan.DecodeJSON([]byte(tc[0]))
		if err != nil || string(got) != tc[1] {
			t.Errorf("%s => %s, %v; want %s", tc[0], got, err, tc[1])
		}
	}
	minimum, maximum := "-9223372036854775808", "9223372036854775807"
	below, above := "-9223372036854775809", "9223372036854775808"
	if strconv.IntSize == 32 {
		minimum, maximum, below, above = "-2147483648", "2147483647", "-2147483649", "2147483648"
	}
	for _, raw := range []string{minimum, maximum} {
		if got, err := plan.DecodeJSON([]byte(raw)); err != nil || string(got) != `"ASAP"` {
			t.Errorf("bound %s => %s, %v", raw, got, err)
		}
	}
	for _, raw := range []string{below, above, `18446744073709551615`, `18446744073709551616`, `1.0`, `1e0`, `1.5`, `true`, `[]`, `{}`, `1 2`} {
		if _, err := plan.DecodeJSON([]byte(raw)); err == nil {
			t.Errorf("accepted %s", raw)
		}
	}
	domain := plan.InputDomain()
	if !domain.OpenIntegers || !domain.OpenStrings || string(domain.Minimum) != minimum || string(domain.Maximum) != maximum || len(domain.StringAlternatives) != 0 {
		t.Fatalf("domain %+v", domain)
	}
	if got := plan.WireValues(false); !reflect.DeepEqual(got, []any{"ASAP", "SPEND_EVENLY"}) {
		t.Fatal(got)
	}
}

func TestScalarStringFallbackIndependentPolicies(t *testing.T) {
	for _, tc := range []struct {
		tag       reflect.StructTag
		good, bad string
	}{
		{`choice:"1:A" enumAccept:"scalar" enumUnknownString:"1"`, `"other"`, `99`},
		{`choice:"1:A" enumAccept:"scalar" enumUnknownInteger:"1"`, `99`, `"other"`},
	} {
		p, err := ParseEnum(tc.tag, reflect.TypeOf(""))
		if err != nil {
			t.Fatal(err)
		}
		if _, err = p.DecodeJSON([]byte(tc.good)); err != nil {
			t.Fatal(err)
		}
		if _, err = p.DecodeJSON([]byte(tc.bad)); err == nil {
			t.Fatal(tc.bad)
		}
	}
	for _, tag := range []reflect.StructTag{
		`enumUnknownInteger:"0"`,
		`choice:"18446744073709551615:A" enumAccept:"scalar" enumUnknownInteger:"18446744073709551615"`,
		`choice:"1:A" enumUnknownInteger:"1"`,
		`choice:"1:A" enumAccept:"scalar" enumUnknownInteger:"0"`,
		`choice:"1:A" enumAccept:"scalar" enumUnknownString:"0"`,
		`choice:"1:A" enumAccept:"scalar" enumUnknownInteger:"1.0"`,
		`choice:"1:A" enumAccept:"scalar" enumUnknownInteger:"1" enumInteger:"passthrough"`,
		`choice:"1:A" enumAccept:"scalar" enumStyle:"ordinal" enumUnknownInteger:"1"`,
	} {
		if _, err := ParseEnum(tag, reflect.TypeOf("")); err == nil {
			t.Errorf("accepted %s", tag)
		}
	}
}

func TestScalarStringFallbackDailyBudget(t *testing.T) {
	p, err := ParseEnum(`choice:"1:FIXED_BUDGET" choice:"2:AUTO_BUDGET" enumAccept:"scalar" enumCase:"upper" enumUnknownString:"1" enumUnknownInteger:"1"`, reflect.TypeOf(""))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range [][2]string{{`2`, `"AUTO_BUDGET"`}, {`3`, `"FIXED_BUDGET"`}, {`"auto_budget"`, `"AUTO_BUDGET"`}, {`"2"`, `"FIXED_BUDGET"`}, {`"other"`, `"FIXED_BUDGET"`}, {`null`, `null`}} {
		got, err := p.DecodeJSON([]byte(tc[0]))
		if err != nil || string(got) != tc[1] {
			t.Errorf("%s => %s %v", tc[0], got, err)
		}
	}
}

func TestScalarFallbackKnownAliasPrecedence(t *testing.T) {
	p, err := ParseEnum(`choice:"0:A" choice:"1:B" enumAccept:"scalar" enumUnknownString:"0" enumUnknownInteger:"0" enumAlias:"1:alias"`, reflect.TypeOf(""))
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{`1`, `"B"`, `"alias"`} {
		got, err := p.DecodeJSON([]byte(raw))
		if err != nil || string(got) != `"B"` {
			t.Fatalf("%s => %s %v", raw, got, err)
		}
	}
}

func TestScalarIntegerFallback(t *testing.T) {
	tag := reflect.StructTag(`choice:"1:FIXED_BUDGET" choice:"2:AUTO_BUDGET" enumAccept:"scalar" enumCase:"upper" enumUnknownString:"1" enumUnknownInteger:"1"`)
	for _, typ := range []reflect.Type{reflect.TypeOf(int(0)), reflect.TypeOf(int8(0))} {
		p, err := ParseEnum(tag, typ)
		if err != nil {
			t.Fatal(err)
		}
		for _, tc := range [][2]string{{`2`, `2`}, {`1`, `1`}, {`0`, `1`}, {`-1`, `1`}, {`3`, `1`}, {`"AUTO_BUDGET"`, `2`}, {`"auto_budget"`, `2`}, {`"other"`, `1`}, {`"2"`, `1`}, {`null`, `null`}} {
			got, err := p.DecodeJSON([]byte(tc[0]))
			if err != nil || string(got) != tc[1] {
				t.Errorf("%s %s => %s %v", typ, tc[0], got, err)
			}
		}
		d := p.InputDomain()
		if !d.OpenIntegers || !d.OpenStrings {
			t.Fatal(d)
		}
		if typ.Kind() == reflect.Int8 {
			if d.Minimum != "-128" || d.Maximum != "127" {
				t.Fatal(d)
			}
			for _, raw := range []string{`128`, `-129`} {
				if _, err := p.DecodeJSON([]byte(raw)); err == nil {
					t.Errorf("accepted %s for int8", raw)
				}
			}
		}
		if got := p.WireValues(false); !reflect.DeepEqual(got, []any{int64(1), int64(2)}) {
			t.Fatalf("output domain %v", got)
		}
	}
	for _, tag := range []reflect.StructTag{
		`choice:"1:A" enumAccept:"scalar" enumUnknownInteger:"0"`,
		`choice:"1:A" enumAccept:"scalar" enumUnknownInteger:"1" enumInteger:"passthrough"`,
		`choice:"1:A" enumAccept:"scalar" enumUnknownInteger:"1" enumStyle:"literal"`,
	} {
		if _, err := ParseEnum(tag, reflect.TypeOf(int(0))); err == nil {
			t.Errorf("accepted %s", tag)
		}
	}
}
