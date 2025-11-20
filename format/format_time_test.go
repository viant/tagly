package format

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/tagly/format/text"
	"testing"
	"time"
)

func TestTag_FormatTime(t *testing.T) {
	base := time.Date(2023, 7, 5, 12, 34, 56, 0, time.FixedZone("X", -7*3600))

	var testCases = []struct {
		description string
		tag         Tag
		input       time.Time
		expect      string
	}{
		{
			description: "default layout RFC3339 in UTC (no timezone set)",
			tag:         Tag{},
			input:       base.In(time.UTC),
			expect:      base.In(time.UTC).Format(time.RFC3339),
		},
		{
			description: "custom layout applied",
			tag:         Tag{TimeLayout: "2006-01-02 15:04:05"},
			input:       base,
			expect:      base.Format("2006-01-02 15:04:05"),
		},
		{
			description: "timezone = UTC coerces to UTC",
			tag:         Tag{Timezone: "UTC"},
			input:       base, // non-UTC input
			expect:      base.In(time.UTC).Format(time.RFC3339),
		},
		{
			description: "invalid timezone leaves time unchanged",
			tag:         Tag{Timezone: "Mars/Curiosity", TimeLayout: time.RFC3339},
			input:       base,
			expect:      base.Format(time.RFC3339),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			// copy to avoid pointer sharing in adjustTimezone
			in := tc.input
			got := tc.tag.FormatTime(&in)
			assert.Equal(t, tc.expect, got, tc.description)
		})
	}
}

func TestTag_CaseFormatName(t *testing.T) {
	var testCases = []struct {
		description string
		tag         Tag
		defaultTo   text.CaseFormat
		expect      string
	}{
		{
			description: "explicit case format takes precedence",
			tag:         Tag{Name: "AbcXyz", CaseFormat: string(text.CaseFormatLowerCamel)},
			defaultTo:   text.CaseFormatLowerUnderscore,
			expect:      "abcXyz",
		},
		{
			description: "default case format used when none provided",
			tag:         Tag{Name: "AbcXyz"},
			defaultTo:   text.CaseFormatLowerUnderscore,
			expect:      "abc_xyz",
		},
		{
			description: "ignore case formatter with '-' returns original name",
			tag:         Tag{Name: "AbcXyz", CaseFormat: "-"},
			defaultTo:   text.CaseFormatLowerUnderscore,
			expect:      "AbcXyz",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			got := tc.tag.CaseFormatName(tc.defaultTo)
			assert.Equal(t, tc.expect, got)

			// ensure cached formatter does not change result on subsequent calls
			got2 := tc.tag.CaseFormatName(tc.defaultTo)
			assert.Equal(t, tc.expect, got2)
		})
	}
}
