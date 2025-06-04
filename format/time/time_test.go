package time

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParse(t *testing.T) {
	var testCases = []struct {
		description string
		layout      string
		input       string
	}{

		{
			description: "go time with timezone",
			input:       "2025-06-04T11:54:28.977155-07:00",
		},
		{
			description: "go time",
			input:       "2025-06-04T18:08:30.80335",
		},
		{
			description: "iso time",
			input:       "2023-01-02 01:22:19",
		},
		{
			description: "rfc time",
			input:       "2023-01-02T01:22:19",
		},
		{
			description: "date",
			input:       "2023-01-02",
		},
	}

	//  "message": "failed to unmarshal Data[0].CreatedTime, parsing time \"2025-06-04T18:08:30.80335\" as \"2006-01-02-07:00\": cannot parse \"T18:08:30.80335\" as \"-07:00\""
	//

	for _, testCase := range testCases {
		ts, err := Parse(testCase.layout, testCase.input)
		assert.Nil(t, err, testCase.description)
		fmt.Printf("%s\n", ts.String())
	}
}
