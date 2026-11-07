package choice

import (
	"encoding/json"
	"reflect"
	"regexp"
	"testing"
	"unicode/utf8"
)

type pacingAlias string

const pacingAliasTag reflect.StructTag = `choice:"0:ASAP" choice:"1:SPEND_EVENLY" enumAccept:"scalar" enumStyle:"literal" enumAlias:"0:0" enumAlias:"1:1" enumAlias:"1:Even" enumCase:"fold"`

func TestScalarAliasesDecodeToCanonicalStorage(t *testing.T) {
	for _, typ := range []reflect.Type{reflect.TypeOf(""), reflect.TypeOf(pacingAlias("")), reflect.TypeOf((*pacingAlias)(nil))} {
		t.Run(typ.String(), func(t *testing.T) {
			plan, err := ParseEnum(pacingAliasTag, typ)
			if err != nil || plan == nil {
				t.Fatalf("parse: plan=%v error=%v", plan, err)
			}
			for _, tc := range []struct{ raw, want string }{
				{`0`, `"ASAP"`}, {`1`, `"SPEND_EVENLY"`},
				{`"0"`, `"ASAP"`}, {`"1"`, `"SPEND_EVENLY"`},
				{`"ASAP"`, `"ASAP"`}, {`"asap"`, `"ASAP"`},
				{`"Even"`, `"SPEND_EVENLY"`}, {`"eVeN"`, `"SPEND_EVENLY"`},
				{`"spend_evenly"`, `"SPEND_EVENLY"`},
				{`"ſPEND_EVENLY"`, `"SPEND_EVENLY"`},
				{`null`, `null`},
			} {
				t.Run(tc.raw, func(t *testing.T) {
					got, err := plan.DecodeJSON([]byte(tc.raw))
					if err != nil || string(got) != tc.want {
						t.Fatalf("got %s, error=%v; want %s", got, err, tc.want)
					}
				})
			}
			for _, raw := range []string{`2`, `-1`, `1.5`, `1.0`, `1e0`, `9007199254740993`, `"2"`, `"FRONTLOAD"`, `"Even\n"`, `true`, `{}`, `[]`, `1 0`} {
				if got, err := plan.DecodeJSON([]byte(raw)); err == nil {
					t.Errorf("accepted invalid input %s as %s", raw, got)
				}
			}
			if got := plan.WireValues(false); !reflect.DeepEqual(got, []any{"ASAP", "SPEND_EVENLY"}) {
				t.Fatalf("input aliases leaked into output choices: %v", got)
			}
			encoded, err := plan.Encode(pacingAlias("SPEND_EVENLY"))
			if err != nil || encoded != "SPEND_EVENLY" {
				t.Fatalf("canonical output: %v, %v", encoded, err)
			}
		})
	}
}

func TestScalarAliasesRejectInvalidDeclarations(t *testing.T) {
	for _, tag := range []reflect.StructTag{
		`enumAlias:"0:now"`,
		`enumCase:"fold"`,
		`choice:"0:ASAP" enumAccept:"scalar" enumAlias:"2:Even"`,
		`choice:"0:ASAP" enumAccept:"scalar" enumAlias:"Even"`,
		`choice:"0:ASAP" enumAccept:"literal" enumAlias:"0:now"`,
		`choice:"0:ASAP" enumAccept:"ordinal" enumCase:"fold"`,
		`choice:"0:ASAP" enumAccept:"scalar" enumCase:"guess"`,
		`choice:"0:ASAP" enumAccept:"scalar" enumStyle:"scalar"`,
		`choice:"0:ASAP" choice:"1:asap" enumAccept:"scalar" enumCase:"fold"`,
		`choice:"0:ASAP" choice:"1:Even" enumAccept:"scalar" enumAlias:"1:asap" enumCase:"fold"`,
		`choice:"0:ASAP" choice:"1:Even" enumAccept:"scalar" enumAlias:"0:now" enumAlias:"1:now"`,
	} {
		t.Run(string(tag), func(t *testing.T) {
			if _, err := ParseEnum(tag, reflect.TypeOf("")); err == nil {
				t.Fatal("accepted invalid alias declaration")
			}
		})
	}
}

func TestScalarAliasesRequireExplicitStringAliases(t *testing.T) {
	plan, err := ParseEnum(`choice:"0:ASAP" enumAccept:"scalar"`, reflect.TypeOf(""))
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{`"0"`, `"asap"`} {
		if _, err := plan.DecodeJSON([]byte(raw)); err == nil {
			t.Fatalf("accepted undeclared string input %s", raw)
		}
	}
	if got, err := plan.DecodeJSON([]byte(`"ASAP"`)); err != nil || string(got) != `"ASAP"` {
		t.Fatalf("exact canonical input: %s %v", got, err)
	}
	domain := plan.InputDomain()
	if domain == nil || !reflect.DeepEqual(domain.Integers, []json.Number{"0"}) || !reflect.DeepEqual(domain.Strings, []string{"ASAP"}) {
		t.Fatalf("unexpected input domain: %+v", domain)
	}
	// A caller cannot mutate the compiled mapping through schema metadata.
	domain.Strings[0] = "changed"
	if plan.InputDomain().Strings[0] != "ASAP" {
		t.Fatal("schema metadata aliases compiled mapping storage")
	}
}

func TestScalarAliasesDoNotTightenLegacyEnums(t *testing.T) {
	for _, tc := range []struct {
		tag reflect.StructTag
		typ reflect.Type
		raw string
	}{
		{`choice:"0:ASAP" enumAccept:"literal"`, reflect.TypeOf(""), `"unknown"`},
		{`choice:"0:ASAP" enumAccept:"ordinal"`, reflect.TypeOf(0), `999`},
	} {
		plan, err := ParseEnum(tc.tag, tc.typ)
		if err != nil {
			t.Fatal(err)
		}
		got, err := plan.DecodeJSON([]byte(tc.raw))
		if err != nil || string(got) != tc.raw || !json.Valid(got) {
			t.Fatalf("legacy input changed: got %s error=%v", got, err)
		}
	}
}

func TestScalarAliasesPreserveDeclaredLargeIntegers(t *testing.T) {
	for _, typ := range []reflect.Type{reflect.TypeOf(""), reflect.TypeOf(uint64(0))} {
		plan, err := ParseEnum(`choice:"18446744073709551615:maximum" enumAccept:"scalar" enumStyle:"literal" enumAlias:"18446744073709551615:max"`, typ)
		if err != nil {
			t.Fatal(err)
		}
		want := `"maximum"`
		if typ.Kind() == reflect.Uint64 {
			want = `18446744073709551615`
		}
		for _, raw := range []string{`18446744073709551615`, `"maximum"`, `"max"`} {
			got, err := plan.DecodeJSON([]byte(raw))
			if err != nil || string(got) != want {
				t.Fatalf("%s input=%s got=%s want=%s error=%v", typ, raw, got, want, err)
			}
		}
		if _, err := plan.DecodeJSON([]byte(`18446744073709551614`)); err == nil {
			t.Fatal("adjacent undeclared integer was accepted")
		}
	}
}

func TestScalarAliasesRejectCustomOwner(t *testing.T) {
	if _, err := ParseEnum(pacingAliasTag, reflect.TypeOf(wireJSONString(""))); err == nil {
		t.Fatal("scalar aliases bypassed custom JSON owner")
	}
}

func TestScalarAliasStringDomainMatchesDecoding(t *testing.T) {
	plan, err := ParseEnum(pacingAliasTag+` enumAlias:"0:" enumAlias:"1:line\n"`, reflect.TypeOf(""))
	if err != nil {
		t.Fatal(err)
	}
	domain := plan.InputDomain()
	for _, tc := range []struct {
		text     string
		accepted bool
	}{
		{"ASAP", true}, {"asap", true}, {"ſpend_evenly", true},
		{"eVeN", true}, {"0", true}, {"1", true}, {"", true},
		{"line\n", true}, {"line\n\n", false}, {"Even\n", false},
		{"ASAP\r\n", false}, {"\n", false}, {"other", false},
	} {
		t.Run(tc.text, func(t *testing.T) {
			matches := false
			for _, alternative := range domain.StringAlternatives {
				pattern, err := regexp.Compile(alternative.Pattern)
				if err != nil {
					t.Fatalf("schema pattern incompatible with Go validator: %v", err)
				}
				if utf8.RuneCountInString(tc.text) == alternative.Length && pattern.MatchString(tc.text) {
					matches = true
				}
			}
			if matches != tc.accepted {
				t.Fatalf("schema accepted=%v, want %v", matches, tc.accepted)
			}
			raw, _ := json.Marshal(tc.text)
			_, err := plan.DecodeJSON(raw)
			if (err == nil) != tc.accepted {
				t.Fatalf("runtime error=%v, want accepted=%v", err, tc.accepted)
			}
		})
	}
}
