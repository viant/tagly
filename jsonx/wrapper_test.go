package jsonx

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestIntegerArrayWrapper(t *testing.T) {
	policy, err := Compile(`jsonx:"wrapper=ids"`, reflect.TypeOf([]int{}))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, raw, want string
		failure         bool
	}{
		{"bare", `[1,2]`, `[1,2]`, false},
		{"bare empty", `[]`, `[]`, false},
		{"bare null", `null`, `null`, false},
		{"wrapped", `{"ids":[1,2]}`, `[1,2]`, false},
		{"wrapped empty", `{"ids":[]}`, `[]`, false},
		{"case folded", `{"IDS":[3]}`, `[3]`, false},
		{"escaped key", `{"i\u0064s":[4]}`, `[4]`, false},
		{"last duplicate", `{"ids":[1],"Ids":[2]}`, `[2]`, false},
		{"null then values", `{"ids":null,"IDS":[2]}`, `[2]`, false},
		{"duplicate null element", `{"ids":[7],"ids":[null]}`, `[7]`, false},
		{"duplicate reused backing array", `{"ids":[1,2],"ids":[3],"ids":[null,null]}`, `[3,2]`, false},
		{"empty clears backing array", `{"ids":[7],"ids":[],"ids":[null]}`, `[0]`, false},
		{"missing", `{}`, ``, true},
		{"wrapped null", `{"ids":null}`, ``, true},
		{"last null", `{"ids":[1],"IDS":null}`, ``, true},
		{"unknown", `{"creativeIds":[1]}`, ``, true},
		{"extra", `{"ids":[1],"other":1}`, ``, true},
		{"nested", `{"ids":{"ids":[1]}}`, ``, true},
		{"wrong earlier duplicate", `{"ids":"wrong","ids":[1]}`, ``, true},
		{"wrong earlier item", `{"ids":["1"],"ids":[2]}`, ``, true},
		{"fraction", `{"ids":[1.5]}`, ``, true},
		{"overflow", `{"ids":[9223372036854775808]}`, ``, true},
		{"trailing document", `{"ids":[1]} {}`, ``, true},
		{"malformed", `{"ids":[1]`, ``, true},
		{"scalar", `1`, ``, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw, omit, err := policy.Apply([]byte(tc.raw))
			if tc.failure {
				if err == nil {
					t.Fatalf("accepted %s => %s", tc.raw, raw)
				}
				return
			}
			if err != nil || omit || string(raw) != tc.want {
				t.Fatalf("got %s omit=%v err=%v; want %s", raw, omit, err, tc.want)
			}
			var values []int
			if err := json.Unmarshal(raw, &values); err != nil {
				t.Fatal(err)
			}
			if tc.want == "[]" && values == nil {
				t.Fatal("empty array became nil")
			}
		})
	}
}

func TestIntegerArrayWrapperIsExplicit(t *testing.T) {
	plain, err := Compile(``, reflect.TypeOf([]int{}))
	if err != nil {
		t.Fatal(err)
	}
	raw := []byte(`{"ids":[1]}`)
	got, _, err := plain.Apply(raw)
	if err != nil || string(got) != string(raw) {
		t.Fatalf("untagged behavior changed: %s %v", got, err)
	}
	other, err := Compile(`jsonx:"wrapper=values"`, reflect.TypeOf([]int{}))
	if err != nil {
		t.Fatal(err)
	}
	got, _, err = other.Apply([]byte(`{"VALUES":[7]}`))
	if err != nil || string(got) != `[7]` {
		t.Fatalf("declared member ignored: %s %v", got, err)
	}
	for _, rule := range []string{"wrapper=", "wrapper=ids,wrapper=other", "wrapper=ids,objectToSlice", "wrapper=ids,commaStringToSlice"} {
		if _, err := Compile(reflect.StructTag(`jsonx:"`+rule+`"`), reflect.TypeOf([]int{})); err == nil {
			t.Errorf("accepted invalid policy %s", rule)
		}
	}
	for _, typ := range []reflect.Type{reflect.TypeOf(1), reflect.TypeOf([]string{}), reflect.TypeOf([]struct{ ID int }{}), reflect.TypeOf((*[]int)(nil))} {
		if _, err := Compile(`jsonx:"wrapper=ids"`, typ); err == nil {
			t.Errorf("accepted unsupported target %v", typ)
		}
	}
	_, _, err = other.Apply([]byte(`{}`))
	if err == nil || !strings.Contains(err.Error(), "values") {
		t.Fatalf("missing member error must identify declared name: %v", err)
	}
}

type wrapperJSONInt int

func (*wrapperJSONInt) UnmarshalJSON([]byte) error { return nil }

type wrapperTextInt int

func (wrapperTextInt) MarshalText() ([]byte, error) { return []byte("1"), nil }

type wrapperInts []int

func TestIntegerArrayWrapperDomainAndOwnership(t *testing.T) {
	for _, typ := range []reflect.Type{reflect.TypeOf([]wrapperJSONInt{}), reflect.TypeOf([]wrapperTextInt{})} {
		if _, err := Compile(`jsonx:"wrapper=ids"`, typ); err == nil {
			t.Fatalf("custom ownership accepted: %v", typ)
		}
	}
	policy, err := Compile(`jsonx:"wrapper=keys,null=omit"`, reflect.TypeOf(wrapperInts{}))
	if err != nil {
		t.Fatal(err)
	}
	domain := policy.InputDomain()
	if domain.Wrapper != "keys" || !domain.Array || !domain.Null || domain.Object {
		t.Fatalf("wrapper confused with record shape: %+v", domain)
	}
	if _, omit, err := policy.Apply([]byte(`null`)); err != nil || !omit {
		t.Fatalf("bare null omission: %v %v", omit, err)
	}
	if _, _, err := policy.Apply([]byte(`{"keys":null}`)); err == nil {
		t.Fatal("wrapped null must remain invalid with null=omit")
	}
	if got, _, err := policy.Apply([]byte(`{"keys":[]}`)); err != nil || string(got) != "[]" {
		t.Fatalf("named integer slice: %s %v", got, err)
	}
}
