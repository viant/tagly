package format

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTag_ParseTime(t *testing.T) {
	var testCases = []struct {
		description string
		layout      string
		input       string
	}{
		{
			description: "empty layout defaults to RFC3339 (nano if needed)",
			layout:      "",
			input:       "2025-06-04T18:08:30.80335",
		},
		{
			description: "date only",
			layout:      "2006-01-02",
			input:       "2023-01-02",
		},
		{
			description: "rfc3339 with timezone",
			layout:      "2006-01-02T15:04:05Z07:00",
			input:       "2025-06-04T11:54:28.977155-07:00",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			tag := &Tag{TimeLayout: tc.layout}
			ts, err := tag.ParseTime(tc.input)
			assert.Nil(t, err)
			assert.True(t, ts.Year() > 0)
		})
	}
}
