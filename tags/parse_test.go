package tags

import (
	"reflect"
	"testing"
)

func TestParseStrictAndLegacyPrefix(t *testing.T) {
	for _, tc := range []struct {
		source string
		count  int
		fail   bool
	}{
		{`json:"name" sqlx:"NAME"`, 2, false},
		{` json:"" `, 1, false},
		{`predicate:"a" predicate:"b"`, 2, false},
		{`validate:"required" trailing`, 1, true},
		{`validate:required`, 0, true},
		{`json:"name" bad:"unterminated`, 1, true},
		{`json:"name" bad:"\x"`, 1, true},
		{``, 0, false},
	} {
		t.Run(tc.source, func(t *testing.T) {
			legacy := NewTags(tc.source)
			if len(legacy) != tc.count {
				t.Fatalf("legacy=%+v", legacy)
			}
			got, err := Parse(tc.source)
			if (err != nil) != tc.fail {
				t.Fatalf("err=%v", err)
			}
			if tc.fail {
				if got != nil {
					t.Fatal("strict returned partial tags")
				}
				return
			}
			if !reflect.DeepEqual(got, legacy) {
				t.Fatalf("got=%+v legacy=%+v", got, legacy)
			}
			again, err := Parse(got.Literal())
			if err != nil || !reflect.DeepEqual(got, again) {
				t.Fatalf("roundtrip=%+v err=%v", again, err)
			}
		})
	}
	if NewTags(`json:""`).Stringify() != "" {
		t.Fatal("changed legacy empty-tag omission")
	}
	if NewTags(`json:""`).Literal() != `json:""` {
		t.Fatal("lost explicit empty value")
	}
}
