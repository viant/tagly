package jsonx

import (
	"reflect"
	"testing"
)

func TestNumberToIntDomainAndConversions(t *testing.T) {
	target := reflect.TypeOf((*int)(nil))
	p, err := Compile(reflect.StructTag(`jsonx:"numberToInt,duplicate=last"`), target)
	if err != nil {
		t.Fatal(err)
	}
	if !p.LastDuplicateWins() || !reflect.DeepEqual(p.InputDomain(), InputDomain{Number: true, Null: true}) {
		t.Fatal("expected duplicate selection and a nullable numeric input domain")
	}
	for _, tc := range []struct{ raw, want string }{
		{"1", "1"}, {"1.5", "1"}, {"-1.5", "-1"}, {"10e-1", "1"}, {"0.15e1", "1"},
		{"9223372036854775807", "9223372036854775807"}, {"-9223372036854775808", "-9223372036854775808"},
		{"18446744073709551615", "-1"}, {"null", "null"},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			got, omit, err := p.Apply([]byte(tc.raw))
			if err != nil || omit || string(got) != tc.want {
				t.Fatalf("got %s, omitted=%v, error=%v; want %s", got, omit, err, tc.want)
			}
		})
	}
	for _, raw := range []string{`"1"`, "true", "1e999", "18446744073709551616", "-9223372036854775809"} {
		if _, _, err := p.Apply([]byte(raw)); err == nil {
			t.Errorf("accepted %s", raw)
		}
	}
	if _, err := Compile(reflect.StructTag(`jsonx:"numberToInt"`), reflect.TypeOf(0)); err == nil {
		t.Error("accepted a non-pointer int")
	}
	if _, err := Compile(reflect.StructTag(`jsonx:"numberToInt,null=omit"`), target); err == nil {
		t.Error("accepted omitted null presence")
	}
}

func TestDiscardedMemberNumericRange(t *testing.T) {
	for _, tc := range []struct {
		value   string
		invalid bool
	}{
		{`"1e999"`, false}, {"null", false}, {"false", false}, {"  1e-1 ", false}, {"-1.5", false},
		{"18446744073709551615", false}, {"1e999", true}, {"18446744073709551616", true},
		{`{"a":["1e999",{"b":18446744073709551615}]}`, false},
		{`[{"n":[1,18446744073709551616]}]`, true},
	} {
		t.Run(tc.value, func(t *testing.T) {
			err := ValidateDiscardedNumbers([]byte(tc.value))
			if (err != nil) != tc.invalid {
				t.Fatalf("error=%v, want invalid=%v", err, tc.invalid)
			}
		})
	}
}
