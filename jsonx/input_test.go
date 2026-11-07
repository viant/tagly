package jsonx

import (
	"encoding/json"
	"reflect"
	"strconv"
	"testing"
)

func TestInputConversions(t *testing.T) {
	cases := []struct {
		rule      string
		typ       reflect.Type
		raw, want string
		fail      bool
	}{
		{"intToBool", reflect.TypeOf(false), `0`, `false`, false},
		{"intToBool", reflect.TypeOf(false), `1`, `true`, false},
		{"intToBool", reflect.TypeOf(false), `true`, `true`, false},
		{"intToBool", reflect.TypeOf(false), `null`, `null`, false},
		{"intToBool", reflect.TypeOf(false), `2`, ``, true},
		{"intToBool", reflect.TypeOf(false), `"1"`, ``, true},
		{"intToBool", reflect.TypeOf(false), `1.0`, ``, true},
		{"intToBool", reflect.TypeOf(false), `1e0`, ``, true},
		{"numberToString", reflect.TypeOf(""), `9007199254740993`, `"9007199254740993"`, false},
		{"numberToString", reflect.TypeOf(""), `1.50`, `"1.50"`, false},
		{"numberToString", reflect.TypeOf(""), `1e1000`, `"1e1000"`, false},
		{"numberToString", reflect.TypeOf(""), `-0`, `"-0"`, false},
		{"numberToString", reflect.TypeOf(""), `"  x  "`, `"  x  "`, false},
		{"numberToString", reflect.TypeOf(""), `true`, ``, true},
		{"numberToString", reflect.TypeOf(""), `01`, ``, true},
		{"commaStringToSlice", reflect.TypeOf([]string{}), `"A, B ,,C"`, `["A","B","C"]`, false},
		{"commaStringToSlice", reflect.TypeOf([]string{}), `" , "`, `null`, false},
		{"commaStringToSlice", reflect.TypeOf([]string{}), `[]`, `[]`, false},
		{"commaStringToSlice", reflect.TypeOf([]string{}), `[" B ",null]`, `[" B ",null]`, false},
		{"commaStringToSlice", reflect.TypeOf([]string{}), `[1]`, ``, true},
		{"boolIntToYN", reflect.TypeOf(""), `false`, `"N"`, false},
		{"boolIntToYN", reflect.TypeOf(""), `true`, `"Y"`, false},
		{"boolIntToYN", reflect.TypeOf(""), `-2`, `"Y"`, false},
		{"boolIntToYN", reflect.TypeOf(""), `0`, `"N"`, false},
		{"boolIntToYN", reflect.TypeOf(""), `"y"`, `"Y"`, false},
		{"boolIntToYN", reflect.TypeOf(""), `"n"`, `"N"`, false},
		{"boolIntToYN", reflect.TypeOf(""), `"1"`, `"Y"`, false},
		{"boolIntToYN", reflect.TypeOf(""), `"0"`, `"N"`, false},
		{"boolIntToYN", reflect.TypeOf(""), `""`, `""`, false},
		{"boolIntToYN", reflect.TypeOf(""), `" Y "`, ``, true},
		{"boolIntToYN", reflect.TypeOf(""), `1e0`, ``, true},
		{"boolIntToYN", reflect.TypeOf(""), `9223372036854775808`, ``, true},
	}
	for _, tc := range cases {
		t.Run(tc.rule+tc.raw, func(t *testing.T) {
			p, err := Compile(reflect.StructTag(`jsonx:"`+tc.rule+`"`), tc.typ)
			if err != nil {
				t.Fatal(err)
			}
			got, omit, err := p.Apply([]byte(tc.raw))
			if tc.fail {
				if err == nil {
					t.Fatalf("accepted %s => %s", tc.raw, got)
				}
				return
			}
			if err != nil || omit || string(got) != tc.want {
				t.Fatalf("got %s omit=%v err=%v want %s", got, omit, err, tc.want)
			}
		})
	}
}
func TestInputPolicyScopeAndDomain(t *testing.T) {
	for _, rule := range []string{"intToBool", "numberToString", "commaStringToSlice", "boolIntToYN"} {
		if _, err := Compile(reflect.StructTag(`jsonx:"`+rule+`"`), reflect.TypeOf(1.5)); err == nil {
			t.Fatalf("invalid destination accepted: %s", rule)
		}
	}
	if _, err := Compile(`jsonx:"numberToString,boolIntToYN"`, reflect.TypeOf("")); err == nil {
		t.Fatal("conflicting rules accepted")
	}
	p, err := Compile(`jsonx:"boolIntToYN,null=omit"`, reflect.TypeOf(""))
	if err != nil {
		t.Fatal(err)
	}
	_, omit, err := p.Apply([]byte(`null`))
	if !omit || err != nil {
		t.Fatalf("null omission: %v %v", omit, err)
	}
	d := p.InputDomain()
	if !d.Boolean || !d.Integer || len(d.StringValues) != 7 || !d.Null {
		t.Fatalf("wrong domain: %+v", d)
	}
	if strconv.IntSize == 64 && (string(*d.Minimum) != "-9223372036854775808" || string(*d.Maximum) != "9223372036854775807") {
		t.Fatalf("inexact bounds: %+v", d)
	}
	for _, raw := range []string{string(*d.Minimum), string(*d.Maximum)} {
		if _, _, err := p.Apply([]byte(raw)); err != nil {
			t.Fatal(err)
		}
	}
	encoded, _ := json.Marshal(d)
	if !json.Valid(encoded) {
		t.Fatal("invalid domain JSON")
	}
}

func TestInputDomainBranchesAreIndependent(t *testing.T) {
	for _, tc := range []struct {
		rule  string
		typ   reflect.Type
		check func(InputDomain) bool
	}{
		{"intToBool", reflect.TypeOf(false), func(d InputDomain) bool { return d.Boolean && !d.Integer && len(d.IntegerValues) == 2 }},
		{"numberToString", reflect.TypeOf(""), func(d InputDomain) bool { return d.String && d.Number && !d.Integer }},
		{"commaStringToSlice", reflect.TypeOf([]string{}), func(d InputDomain) bool { return d.String && d.Array }},
	} {
		p, err := Compile(reflect.StructTag(`jsonx:"`+tc.rule+`"`), tc.typ)
		if err != nil {
			t.Fatal(err)
		}
		if d := p.InputDomain(); !tc.check(d) {
			t.Fatalf("%s: %+v", tc.rule, d)
		}
	}
	p, _ := Compile(`jsonx:"boolIntToYN"`, reflect.TypeOf(""))
	a := p.InputDomain()
	*a.Minimum = "0"
	a.StringValues[0] = "wrong"
	b := p.InputDomain()
	if string(*b.Minimum) == "0" || b.StringValues[0] != "Y" {
		t.Fatal("mutable shared metadata")
	}
}

type ownedString string

func (*ownedString) UnmarshalJSON([]byte) error { return nil }
func TestInputCustomOwner(t *testing.T) {
	if _, err := Compile(`jsonx:"numberToString"`, reflect.TypeOf(ownedString(""))); err == nil {
		t.Fatal("custom owner overridden")
	}
}
