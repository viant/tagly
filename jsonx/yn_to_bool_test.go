package jsonx

import (
	"reflect"
	"testing"
)

func TestYNToBool(t *testing.T) {
	tag := reflect.StructTag(`jsonx:"ynToBool,null=omit"`)
	for _, typ := range []reflect.Type{reflect.TypeOf(false), reflect.TypeOf((*bool)(nil))} {
		p, err := Compile(tag, typ)
		if err != nil {
			t.Fatal(err)
		}
		wantDomain := InputDomain{Boolean: true, Null: true, StringValues: []string{"Y", "y", "N", "n"}}
		if got := p.InputDomain(); !reflect.DeepEqual(got, wantDomain) {
			t.Errorf("%v domain = %+v, want %+v", typ, got, wantDomain)
		}
		for raw, want := range map[string]string{
			`"Y"`: "true", `"y"`: "true", `"N"`: "false", `"n"`: "false",
			`"\u0059"`: "true", `true`: "true", `false`: "false", `null`: "null",
		} {
			got, omit, err := p.Apply([]byte(raw))
			if err != nil || string(got) != want || omit != (raw == "null") {
				t.Errorf("%v input %s = %s, omit %t, err %v", typ, raw, got, omit, err)
			}
		}
		for _, raw := range []string{`0`, `1`, `1.0`, `"0"`, `"1"`, `""`, `" y"`, `"N "`, `"yes"`, `"true"`, `"false"`, `[]`, `{}`} {
			if _, _, err := p.Apply([]byte(raw)); err == nil {
				t.Errorf("%v accepted %s", typ, raw)
			}
		}
	}
	for _, typ := range []reflect.Type{reflect.TypeOf(""), reflect.TypeOf(0), reflect.TypeOf((**bool)(nil))} {
		if _, err := Compile(tag, typ); err == nil {
			t.Errorf("accepted destination %v", typ)
		}
	}
}
