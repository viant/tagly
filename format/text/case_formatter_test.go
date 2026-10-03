package text

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCase_Format(t *testing.T) {
	var useCases = []struct {
		description string
		from        CaseFormat
		to          CaseFormat
		input       string
		expect      string
	}{

		{
			description: "lower camel to upper underscore",
			input:       "abcXyzId",
			from:        CaseFormatLowerCamel,
			to:          CaseFormatUpperUnderscore,
			expect:      "ABC_XYZ_ID",
		},
		{
			description: "upper camel to upper underscore",
			input:       "AbcXyzId",
			from:        CaseFormatUpperCamel,
			to:          CaseFormatUpperUnderscore,
			expect:      "ABC_XYZ_ID",
		},
		{
			description: "upper underscore to upper camel ",
			input:       "ABC_XYZ_ID",
			from:        CaseFormatUpperUnderscore,
			to:          CaseFormatUpperCamel,
			expect:      "AbcXyzId",
		},
		{
			description: "upper underscore to sentence",
			input:       "ABC_XYZ_ID",
			from:        CaseFormatUpperUnderscore,
			to:          CaseFormatSentence,
			expect:      "Abc xyz id",
		},
		{
			description: " dash",
			input:       "abcXyzID",
			from:        CaseFormatLowerCamel,
			to:          CaseFormatDash,
			expect:      "abc-Xyz-ID",
		},
		{
			description: "lower camel dash",
			input:       "abcXyzID",
			from:        CaseFormatLowerCamel,
			to:          CaseFormatLowerUnderscore,
			expect:      "abc_xyz_id",
		},
		{
			description: "upper camel to lower camel",
			input:       "Vendor",
			from:        CaseFormatUpperCamel,
			to:          NewCaseFormat("lc"),
			expect:      "vendor",
		},
		{
			description: "number",
			input:       "Peer39CustomAdvanced",
			from:        CaseFormatUpperUnderscore,
			to:          NewCaseFormat("lc"),
			expect:      "peer39CustomAdvanced",
		},
	}

	for _, useCase := range useCases {
		formatter := useCase.from.To(useCase.to)
		actual := formatter.Format(useCase.input)
		assert.EqualValues(t, useCase.expect, actual, useCase.description)
		assert.EqualValues(t, useCase.expect, useCase.from.Format(useCase.input, useCase.to), useCase.description)

	}

}

func TestCase_FormatNumericSegments(t *testing.T) {
	patterns := []struct {
		name   string
		prefix string
		suffix string
		want   func(string) string
	}{
		{"sample window", "SampleSeen_", "Day", func(n string) string { return "sampleSeen" + n + "Day" }},
		{"estimated window", "EstimatedFullSeen_", "DayFromRatio", func(n string) string { return "estimatedFullSeen" + n + "DayFromRatio" }},
		{"average window", "AvgHourSeenRatio_", "Day", func(n string) string { return "avgHourSeenRatio" + n + "Day" }},
		{"leading", "", "DaySampleSeen", func(n string) string { return n + "DaySampleSeen" }},
		{"embedded", "SampleSeen", "Day", func(n string) string { return "sampleSeen" + n + "Day" }},
		{"trailing", "SampleSeen", "", func(n string) string { return "sampleSeen" + n }},
		{"trailing after underscore", "SampleSeen_", "", func(n string) string { return "sampleSeen" + n }},
		{"between underscores", "SampleSeen_", "_Day", func(n string) string { return "sampleSeen" + n + "Day" }},
	}
	for _, pattern := range patterns {
		t.Run(pattern.name, func(t *testing.T) {
			seen := make(map[string]string)
			for _, number := range []string{"0", "1", "7", "10", "14", "30", "0014", "12345678901234567890", "١٤"} {
				input := pattern.prefix + number + pattern.suffix
				t.Run(number, func(t *testing.T) {
					want := pattern.want(number)
					actual := DetectCaseFormat(input).Format(input, NewCaseFormat("lc"))
					assert.Equal(t, want, actual)
					if previous, ok := seen[actual]; ok {
						t.Errorf("%q and %q both format as %q", previous, input, actual)
					}
					seen[actual] = input
					// Explicit source formats must preserve numbers too.
					for _, from := range []CaseFormat{CaseFormatUpperCamel, CaseFormatLowerCamel, CaseFormatUpperUnderscore} {
						assert.Equal(t, want, from.To(CaseFormatLowerCamel).Format(input), "source format: %s", from)
					}
				})
			}
		})
	}
}

func TestCase_FormatLowerCamelBoundaries(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  string
	}{
		{"SampleSeen_1Day_14Day", "sampleSeen1Day14Day"},
		{"sample_seen_14_day", "sampleSeen14Day"},
		{"SampleSeen 14Day", "sampleSeen14Day"},
		{"SampleSeen_14.5Day", "sampleSeen14.5Day"},
		{"SampleSeen_.14Day", "sampleSeen.14Day"},
		{"12345678901234567890", "12345678901234567890"},
		{"SampleSeen_Day", "sampleSeenDay"},
		{"SampleSeen_.Day", "sampleSeen_Day"},
		{"SampleSeenDay", "sampleSeenDay"},
		{"sample_seen_day", "sampleSeenDay"},
		{"SAMPLE_SEEN_DAY", "sampleSeenDay"},
		{"Peer39CustomAdvanced", "peer39CustomAdvanced"},
		{"", ""},
	} {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.want, DetectCaseFormat(tc.input).Format(tc.input, NewCaseFormat("lc")))
		})
	}
}
