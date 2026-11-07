package format

import (
	"reflect"
	"testing"
)

func TestInputNamesPreserveCanonicalNameAndExactAliases(t *testing.T) {
	for _, tc := range []struct {
		tag  reflect.StructTag
		want string
	}{
		{`format:"inputNames=metric_25" json:"metric25,omitempty"`, "metric_25"},
		{`format:"inputNames=metric_25|Metric_25" json:"metric25,omitempty"`, "metric_25|Metric_25"},
		{`format:"inputNames=metric value|métrique" json:"metric25,omitempty"`, "metric value|métrique"},
	} {
		parsed, err := Parse(tc.tag, "json")
		if err != nil {
			t.Errorf("%s: %v", tc.tag, err)
			continue
		}
		if parsed.Name != "metric25" || !parsed.Omitempty {
			t.Errorf("canonical output metadata changed: %+v", parsed)
		}
		if parsed.InputNames != tc.want {
			t.Errorf("input aliases not preserved: %+v", parsed)
		}
	}
	if !IsValidTagKey("inputNames") {
		t.Error("inputNames is not recognized as format metadata")
	}
}

func TestInputNamesRejectEmptyAndDuplicateAliases(t *testing.T) {
	for _, tag := range []reflect.StructTag{
		`format:"inputNames="`, `format:"inputNames=|metric_25"`,
		`format:"inputNames=metric_25|"`, `format:"inputNames=metric_25||other"`,
		`format:"inputNames=metric_25|metric_25"`,
	} {
		if _, err := Parse(tag); err == nil {
			t.Errorf("invalid aliases accepted: %s", tag)
		}
	}
}
