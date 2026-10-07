package time

import (
	"testing"
	stdtime "time"
)

func TestParseTimezoneAndPrecision(t *testing.T) {
	for _, tc := range []struct{ name, layout, value, expected string }{
		{"fraction without zone", "", "2025-06-04T18:08:30.80335", "2025-06-04T18:08:30.80335Z"},
		{"fraction with zone", "", "2025-06-04T11:54:28.977155-07:00", "2025-06-04T11:54:28.977155-07:00"},
		{"utc", "", "2025-06-04T18:08:30Z", "2025-06-04T18:08:30Z"},
		{"space without zone", "", "2023-01-02 01:22:19", "2023-01-02T01:22:19Z"},
		{"date", "", "2023-01-02", "2023-01-02T00:00:00Z"},
		{"compact missing zone", "2006-01-02T15:04:05Z0700", "2025-06-04T18:08:30", "2025-06-04T18:08:30Z"},
		{"compact explicit zone", "2006-01-02T15:04:05Z0700", "2025-06-04T18:08:30+0530", "2025-06-04T18:08:30+05:30"},
		{"hour missing zone", "2006-01-02T15:04:05Z07", "2025-06-04T18:08:30", "2025-06-04T18:08:30Z"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Parse(tc.layout, tc.value)
			if err != nil {
				t.Fatal(err)
			}
			if actual := got.Format(stdtime.RFC3339Nano); actual != tc.expected {
				t.Fatalf("got %s, want %s", actual, tc.expected)
			}
		})
	}
}
