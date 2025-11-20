package time

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDateFormatToTimeLayout(t *testing.T) {
	var testCases = []struct {
		description string
		in          string
		expect      string
	}{
		{"date only", "YYYY-MM-DD", "2006-01-02"},
		{"single digits", "YYYY-M-D", "2006-1-2"},
		{"date time Z", "YYYY-MM-DDThh:mm:ssZ", "2006-01-02T15:04:05Z07:00"},
		{"millis", "YYYY-MM-DDThh:mm:ss.SSSZ", "2006-01-02T15:04:05.999Z07:00"},
	}
	for _, tc := range testCases {
		got := DateFormatToTimeLayout(tc.in)
		assert.Equal(t, tc.expect, got, tc.description)
	}
}

func TestTimeLayoutToDateFormat(t *testing.T) {
	var testCases = []struct {
		description string
		in          string
		expect      string
	}{
		{"date only", "2006-01-02", "YYYY-MM-DD"},
		{"with zone", "2006-01-02T15:04:05Z07:00", "YYYY-MM-DDThh:mm:ssZ"},
		{"millis", "2006-01-02T15:04:05.999Z07:00", "YYYY-MM-DDThh:mm:ss.SSSZ"},
	}
	for _, tc := range testCases {
		got := TimeLayoutToDateFormat(tc.in)
		assert.Equal(t, tc.expect, got, tc.description)
	}
}
