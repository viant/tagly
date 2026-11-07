package format

import (
	"reflect"
	"testing"
	"time"
)

func TestTimeInputPolicy(t *testing.T) {
	tag, err := Parse(reflect.StructTag(`format:"timeLayout=2006-01-02T15:04:05Z07:00,timeInputLayouts=2006-01-02T15:04:05.999999999Z07:00|2006-01-02T15:04:05|2006-01-02 15:04:05|2006-01-02,timeEpoch=milliseconds,timeEmpty=preserve,timezone=UTC,nullable=true"`))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, value string
		valid       bool
	}{
		{"date", "2026-11-07", true}, {"space", "2026-11-07 08:00:00", true}, {"no zone", "2026-11-07T08:00:00", true}, {"offset", "2026-11-07T08:00:00-08:00", true}, {"invalid", "nonsense", false}, {"trailing", "2026-11-07garbage", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			value, err := tag.ParseTime(tc.value)
			if (err == nil) != tc.valid {
				t.Fatalf("value=%v error=%v", value, err)
			}
			if tc.valid && value.Location() != time.UTC {
				t.Fatalf("not UTC: %v", value)
			}
		})
	}
}

func TestTimePolicyJSONScalars(t *testing.T) {
	tag := &Tag{TimeInputLayouts: time.RFC3339Nano + "|2006-01-02", TimeEpoch: "milliseconds", TimeEmpty: "preserve", Timezone: "UTC"}
	for _, tc := range []struct {
		name, raw string
		action    TimeAction
		value     string
		fail      bool
	}{
		{"epoch", `0`, TimeAssign, "1970-01-01T00:00:00Z", false},
		{"negative", `-1`, TimeAssign, "1969-12-31T23:59:59.999Z", false},
		{"offset", `"2026-11-07T08:00:00-08:00"`, TimeAssign, "2026-11-07T16:00:00Z", false},
		{"fractional seconds", `"2026-11-07T08:00:00.123456789Z"`, TimeAssign, "2026-11-07T08:00:00.123456789Z", false},
		{"zero", `"0001-01-01T00:00:00Z"`, TimeAssign, "0001-01-01T00:00:00Z", false},
		{"null", `null`, TimeNull, "", false}, {"empty", `""`, TimePreserve, "", false},
		{"overflow", `9223372036854775808`, TimeAssign, "", true},
		{"fraction", `1.0`, TimeAssign, "", true}, {"exponent", `1e3`, TimeAssign, "", true},
		{"numeric string", `"123"`, TimeAssign, "", true}, {"bool", `true`, TimeAssign, "", true},
		{"object", `{}`, TimeAssign, "", true}, {"array", `[]`, TimeAssign, "", true},
		{"trailing", `"2026-11-07tail"`, TimeAssign, "", true}, {"bad JSON", `01`, TimeAssign, "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			value, action, err := tag.ParseTimeJSON([]byte(tc.raw))
			if (err != nil) != tc.fail {
				t.Fatalf("error=%v value=%v", err, value)
			}
			if err != nil {
				return
			}
			if action != tc.action {
				t.Fatalf("action=%v want %v", action, tc.action)
			}
			if tc.value != "" && value.Format(time.RFC3339Nano) != tc.value {
				t.Fatalf("value=%v want %s", value, tc.value)
			}
		})
	}
}

func TestTimeInputPolicyRejectsInvalidMetadata(t *testing.T) {
	for _, raw := range []string{`format:"timeEpoch=seconds"`, `format:"timeEmpty=clear"`, `format:"timeInputLayouts=2006-01-02|"`} {
		if _, err := Parse(reflect.StructTag(raw)); err == nil {
			t.Errorf("accepted %s", raw)
		}
	}
}
