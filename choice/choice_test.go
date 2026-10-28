package choice

import (
	"encoding/json"
	"reflect"
	"testing"
)

type mode int64
type label string

func TestOrderedTypedChoices(t *testing.T) {
	for _, tc := range []struct {
		tag  reflect.StructTag
		typ  reflect.Type
		want string
	}{
		{`choice:"3" choice:"1" choice:"2"`, reflect.TypeOf((*int)(nil)).Elem(), `[3,1,2]`},
		{`choice:"z" choice:"a" choice:"m"`, reflect.TypeOf((*label)(nil)).Elem(), `["z","a","m"]`},
		{`choice:"3" choice:"1" choice:"2"`, reflect.TypeOf((*string)(nil)).Elem(), `["3","1","2"]`},
		{`choice:"9007199254740993" choice:"9007199254740994"`, reflect.TypeOf((**mode)(nil)).Elem(), `[9007199254740993,9007199254740994]`},
		{`choice:"false" choice:"true"`, reflect.TypeOf((*bool)(nil)).Elem(), `[false,true]`},
		{`choice:" 3 " choice:"a\"b" choice:"a\\b" choice:""`, reflect.TypeOf((*string)(nil)).Elem(), `[" 3 ","a\"b","a\\b",""]`},
	} {
		c, e := Parse(tc.tag, tc.typ)
		if e != nil {
			t.Fatal(e)
		}
		b, _ := json.Marshal(c.Values)
		if string(b) != tc.want {
			t.Fatalf("%s != %s", b, tc.want)
		}
	}
}
func TestRejectInvalidChoices(t *testing.T) {
	for _, tc := range []struct {
		tag reflect.StructTag
		typ reflect.Type
	}{{`choice:"1" choice:"1"`, reflect.TypeOf((*int)(nil)).Elem()}, {`choice:"1" choice:"01"`, reflect.TypeOf((*int)(nil)).Elem()}, {`choice:"256"`, reflect.TypeOf((*uint8)(nil)).Elem()}, {`choice:"NaN"`, reflect.TypeOf((*float64)(nil)).Elem()}, {`choice:"1.1"`, reflect.TypeOf((*int64)(nil)).Elem()}, {`choice:"true`, reflect.TypeOf((*bool)(nil)).Elem()}, {`choice:"one"`, reflect.TypeOf([]string(nil))}, {`choice:"1"`, reflect.TypeOf((*any)(nil)).Elem()}} {
		if _, e := Parse(tc.tag, tc.typ); e == nil {
			t.Fatalf("accepted %s/%v", tc.tag, tc.typ)
		}
	}
}
func TestNormalizeExactMetadata(t *testing.T) {
	c, e := Normalize(reflect.TypeOf((*int64)(nil)).Elem(), []any{json.Number("9007199254740993"), json.Number("9007199254740994")})
	if e != nil {
		t.Fatal(e)
	}
	if c.Values[0] != int64(9007199254740993) {
		t.Fatal("exact integer lost")
	}
	if _, e = Normalize(reflect.TypeOf((*int64)(nil)).Elem(), []any{float64(9007199254740993)}); e == nil {
		t.Fatal("lossy float accepted")
	}
	if _, e = Normalize(reflect.TypeOf((*int)(nil)).Elem(), []any{"3"}); e == nil {
		t.Fatal("schema numeric string accepted")
	}
	other, _ := Parse(`choice:"2" choice:"1"`, reflect.TypeOf((*int)(nil)).Elem())
	first, _ := Parse(`choice:"1" choice:"2"`, reflect.TypeOf((*int)(nil)).Elem())
	if Equal(first, other) {
		t.Fatal("domain order conflict ignored")
	}
}

type wireTextInt int64

func (wireTextInt) MarshalText() ([]byte, error) { return []byte("custom"), nil }

type wireJSONUint uint64

func (wireJSONUint) MarshalJSON() ([]byte, error) { return []byte(`"custom"`), nil }

type wireJSONFloat float32

func (wireJSONFloat) MarshalJSON() ([]byte, error) { return []byte(`"custom"`), nil }

type wireJSONBool bool

func (wireJSONBool) MarshalJSON() ([]byte, error) { return []byte(`"custom"`), nil }

type wireJSONString string

func (wireJSONString) MarshalJSON() ([]byte, error) { return []byte(`"custom"`), nil }

func TestWireValuesRemoveCustomMarshaling(t *testing.T) {
	for _, tc := range []struct {
		typ  reflect.Type
		tag  reflect.StructTag
		want string
	}{
		{reflect.TypeOf(wireTextInt(0)), `choice:"9007199254740993" choice:"1"`, `[9007199254740993,1]`},
		{reflect.TypeOf(wireJSONUint(0)), `choice:"18446744073709551615" choice:"0"`, `[18446744073709551615,0]`},
		{reflect.TypeOf(wireJSONFloat(0)), `choice:"1.2" choice:"0"`, `[1.2,0]`},
		{reflect.TypeOf(wireJSONBool(false)), `choice:"false" choice:"true"`, `[false,true]`},
		{reflect.TypeOf(wireJSONString("")), `choice:"03" choice:" a "`, `["03"," a "]`},
	} {
		c, err := Parse(tc.tag, tc.typ)
		if err != nil {
			t.Fatal(err)
		}
		wire, err := json.Marshal(c.WireValues())
		if err != nil {
			t.Fatal(err)
		}
		if string(wire) != tc.want {
			t.Fatalf("%v wire=%s want %s", tc.typ, wire, tc.want)
		}
		if reflect.TypeOf(c.Values[0]) != tc.typ {
			t.Fatalf("declared Go identity lost for %v", tc.typ)
		}
		original, err := json.Marshal(c.Values)
		if err != nil {
			t.Fatal(err)
		}
		if string(original) == string(wire) {
			t.Fatalf("custom marshaling probe ineffective for %v", tc.typ)
		}
	}
}
