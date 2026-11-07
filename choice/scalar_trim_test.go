package choice

import (
	"encoding/json"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"unicode"
)

const trimmedChannelTag reflect.StructTag = `choice:"1:DISPLAY" choice:"2:VIDEO" enumAlias:"1:1" enumAlias:"2:2" enumAlias:"2:OLV" enumAccept:"scalar" enumStyle:"ordinal" enumInteger:"passthrough" enumCase:"upper" enumTrim:"space"`

func TestScalarTrimRuntimeAndDefaults(t *testing.T) {
	p, err := ParseEnum(trimmedChannelTag, reflect.TypeOf(int64(0)))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ raw, want string }{
		{`" display "`, `1`}, {`"\tOlV\n"`, `2`}, {`"\u0085\u00a01\u3000"`, `1`},
		{`99`, `99`}, {`-99`, `-99`}, {`null`, `null`}, {`9223372036854775807`, `9223372036854775807`},
	} {
		got, err := p.DecodeJSON([]byte(tc.raw))
		if err != nil || string(got) != tc.want {
			t.Errorf("%s => %s, %v; want %s", tc.raw, got, err, tc.want)
		}
	}
	for _, raw := range []string{`"99"`, `" 99 "`, `" "`, `""`, `true`, `{}`, `[]`, `1.0`, `1e0`, `9223372036854775808`, `"\u180eDISPLAY"`, `"DISPLAY\ufeff"`} {
		if _, err := p.DecodeJSON([]byte(raw)); err == nil {
			t.Errorf("accepted %s", raw)
		}
	}
	plain, err := ParseEnum(reflect.StructTag(strings.ReplaceAll(string(trimmedChannelTag), ` enumTrim:"space"`, "")), reflect.TypeOf(int64(0)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = plain.DecodeJSON([]byte(`" DISPLAY "`)); err == nil {
		t.Fatal("trim leaked into default enum")
	}
}

func TestScalarTrimMetadata(t *testing.T) {
	for _, tag := range []reflect.StructTag{
		`enumTrim:"space"`,
		`choice:"1:A" enumTrim:"space"`,
		`choice:"1:A" enumAccept:"scalar" enumTrim:"all"`,
		`choice:"1:A" enumAccept:"scalar" enumTrim:""`,
		`choice:"1:A" choice:"2: A " enumAccept:"scalar" enumTrim:"space"`,
		`choice:"1:A" choice:"2:B" enumAlias:"2: A " enumAccept:"scalar" enumTrim:"space"`,
	} {
		if _, err := ParseEnum(tag, reflect.TypeOf(int(0))); err == nil {
			t.Errorf("accepted invalid metadata %s", tag)
		}
	}
}

func TestScalarTrimDomainMatchesUnicodeWhitespace(t *testing.T) {
	p, err := ParseEnum(trimmedChannelTag, reflect.TypeOf(int64(0)))
	if err != nil {
		t.Fatal(err)
	}
	domain := p.InputDomain()
	for _, branch := range domain.StringAlternatives {
		if !branch.VariableLength {
			t.Fatal("trim schema must not retain fixed maximum length")
		}
	}
	for _, value := range []string{"DISPLAY", " display ", "\tOlV\n", "\u0085\u00a0VIDEO\u3000", "\u1680\u2000\u200a2\u2028\u2029\u202f\u205f", "\u180eDISPLAY", "DISPLAY\ufeff", "99", " 99 ", ""} {
		raw, _ := json.Marshal(value)
		_, runtimeErr := p.DecodeJSON(raw)
		admitted := false
		for _, branch := range domain.StringAlternatives {
			pattern, err := regexp.Compile(branch.Pattern)
			if err != nil {
				t.Fatal(err)
			}
			if pattern.MatchString(value) {
				admitted = true
			}
		}
		if admitted != (runtimeErr == nil) {
			t.Errorf("schema/runtime mismatch %q: admitted=%v error=%v", value, admitted, runtimeErr)
		}
	}
	for r := rune(0); r <= 0x3000; r++ {
		if !unicode.IsSpace(r) {
			continue
		}
		value := string(r) + "DISPLAY" + string(r)
		matched := false
		for _, branch := range domain.StringAlternatives {
			if regexp.MustCompile(branch.Pattern).MatchString(value) {
				matched = true
			}
		}
		if !matched {
			t.Errorf("schema rejects Go whitespace U+%04X", r)
		}
	}
}

func TestScalarTrimPreservesCanonicalLiteralOutput(t *testing.T) {
	p, err := ParseEnum(`choice:"1: A " enumAccept:"scalar" enumTrim:"space"`, reflect.TypeOf(""))
	if err != nil {
		t.Fatal(err)
	}
	got, err := p.DecodeJSON([]byte(`"\tA\n"`))
	if err != nil || string(got) != `" A "` {
		t.Fatalf("canonical literal changed: %s %v", got, err)
	}
	plain, err := ParseEnum(`choice:"1:A" enumAccept:"scalar"`, reflect.TypeOf(int(0)))
	if err != nil {
		t.Fatal(err)
	}
	for _, branch := range plain.InputDomain().StringAlternatives {
		if branch.VariableLength {
			t.Fatal("untagged input lost exact length constraint")
		}
	}
}
