package choice

import (
	"encoding"
	"encoding/json"
	"reflect"
	"testing"
)

func TestScalarValuePreservesBranch(t *testing.T) {
	for _, raw := range []string{`4237`, `4241`, `99999`, `9223372036854775807`, `-9223372036854775808`, `"AD_SERVING"`, `"NOT_A_FEE"`, `""`, `0`, `null`} {
		t.Run(raw, func(t *testing.T) {
			value, err := ParseScalarValue([]byte(raw))
			if err != nil {
				t.Fatal(err)
			}
			wire, err := value.Value()
			if err != nil {
				t.Fatal(err)
			}
			got, err := json.Marshal(wire)
			if err != nil || string(got) != raw {
				t.Fatalf("roundtrip = %s, %v; want %s", got, err, raw)
			}
			// Storage may be decoded by the standard library after native representation conversion.
			stored, err := json.Marshal(value)
			if err != nil {
				t.Fatal(err)
			}
			var restored ScalarValue
			if err := json.Unmarshal(stored, &restored); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(value, restored) {
				t.Fatalf("storage changed: %#v => %#v", value, restored)
			}
		})
	}
}

func TestScalarValueRejectsNonScalarAndOverflow(t *testing.T) {
	for _, raw := range []string{`true`, `{}`, `[]`, `1.0`, `1e2`, `9223372036854775808`, `-9223372036854775809`, `1 2`, `"x" false`, ``, `01`} {
		if _, err := ParseScalarValue([]byte(raw)); err == nil {
			t.Errorf("accepted %s", raw)
		}
	}
	s, i := "x", int64(1)
	if _, err := (ScalarValue{String: &s, Integer: &i}).Value(); err == nil {
		t.Error("accepted ambiguous branches")
	}
}

func TestScalarValueHasNoCustomTransportMethods(t *testing.T) {
	typ := reflect.TypeOf(ScalarValue{})
	for _, iface := range []reflect.Type{
		reflect.TypeOf((*json.Marshaler)(nil)).Elem(), reflect.TypeOf((*json.Unmarshaler)(nil)).Elem(),
		reflect.TypeOf((*encoding.TextMarshaler)(nil)).Elem(), reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem(),
	} {
		if typ.Implements(iface) || reflect.PointerTo(typ).Implements(iface) {
			t.Fatalf("custom transport owner: %s", iface)
		}
	}
}
