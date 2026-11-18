package jsonx_test

import (
	"reflect"
	"testing"

	"github.com/viant/tagly/jsonx"
)

func TestWrapperValidationPublicMessage(t *testing.T) {
	for _, tc := range []struct {
		name, wrapper, raw, want string
	}{
		{"unknown", "ids", `{"ids":[1],"other":2}`, `json: unknown field "other"`},
		{"missing", "ids", `{}`, "ids is required"},
		{"null", "ids", `{"ids":null}`, "ids is required"},
		{"last null", "ids", `{"ids":[1],"IDS":null}`, "ids is required"},
		{"declared name", "values", `{}`, "values is required"},
		{"escaped unknown", "ids", `{"line\nbreak":[]}`, `json: unknown field "line\nbreak"`},
		{"syntax", "ids", `{"ids":`, ""},
		{"trailing", "ids", `{"ids":[]} {}`, ""},
		{"scalar", "ids", `1`, ""},
		{"type", "ids", `{"ids":"private"}`, ""},
		{"item type", "ids", `{"ids":["private"]}`, ""},
		{"fraction", "ids", `{"ids":[1.5]}`, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			policy, err := jsonx.Compile(reflect.StructTag(`jsonx:"wrapper=`+tc.wrapper+`"`), reflect.TypeOf([]int{}))
			if err != nil {
				t.Fatal(err)
			}
			_, _, err = policy.Apply([]byte(tc.raw))
			if err == nil {
				t.Fatal("expected invalid input")
			}
			public, typed := err.(*jsonx.WrapperError)
			message, ok := "", false
			if typed {
				message, ok = public.PublicMessage()
			}
			if ok != (tc.want != "") || message != tc.want {
				t.Fatalf("safe message=(%q,%v), want %q; error=%T", message, ok, tc.want, err)
			}
			if ok && err.Error() != tc.want {
				t.Fatalf("diagnostic changed: %v", err)
			}
		})
	}
}

func TestWrapperValidationZeroValueIsNotPublic(t *testing.T) {
	for _, err := range []*jsonx.WrapperError{nil, {}} {
		if text, ok := err.PublicMessage(); ok || text != "" {
			t.Fatalf("uninitialized error is public: %q, %v", text, ok)
		}
		if err.Error() != "invalid wrapper input" {
			t.Fatalf("uninitialized diagnostic changed: %v", err)
		}
	}
}
