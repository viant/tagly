package tags

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestValues_MatchPairs(t *testing.T) {
	var testCases = []struct {
		description string
		input       string
		expect      map[string]string
	}{

		{

			description: "enclosed",
			input:       ",path,abc,'ewrwe(1,2,3)',3",
			expect: map[string]string{
				"3": "", "abc": "", "'ewrwe(1,2,3)'": "", "path": "",
			},
		},

		{

			description: "enclosed with equal",
			input:       ",path,abc,(1,2=3,3),3",
			expect: map[string]string{
				"3": "", "abc": "", "(1,2=3,3)": "", "path": "",
			},
		},

		{

			description: "mixed",
			input:       "name=abc",
			expect: map[string]string{
				"name": "abc",
			},
		},
		{

			description: "mixed",
			input:       ",omitempty,path=@exclude-ids",
			expect: map[string]string{
				"omitempty": "",
				"path":      "@exclude-ids",
			},
		},
		{

			description: "quoted",
			input:       ",omitempty,path=@exclude-ids,z={1,2,3},v=true",
			expect: map[string]string{
				"omitempty": "",
				"path":      "@exclude-ids",
				"z":         "{1,2,3}",
				"v":         "true",
			},
		},
		{

			description: "name with options",
			input:       "kind=query,in=qp1",
			expect: map[string]string{
				"kind": "query",
				"in":   "qp1",
			},
		},
	}
	for _, testCase := range testCases {
		values := Values(testCase.input)
		actual := map[string]string{}
		err := values.MatchPairs(func(key, value string) error {
			actual[key] = value
			return nil
		})
		assert.Nil(t, err)
		assert.EqualValues(t, testCase.expect, actual, testCase.description)
	}
}

func TestValues_Name(t *testing.T) {
	var testCases = []struct {
		description  string
		input        Values
		expectName   string
		expectValues Values
	}{

		{

			description:  "pairs",
			input:        "p1=1,group=1",
			expectValues: "p1=1,group=1",
			expectName:   "",
		},
		{

			description:  "mixed",
			input:        "p1,group=1",
			expectValues: "group=1",
			expectName:   "p1",
		},
		{

			description:  "mixed 2",
			input:        "p1,arg1",
			expectValues: "arg1",
			expectName:   "p1",
		},
	}
	for _, testCase := range testCases {
		actual, values := testCase.input.Name()
		assert.EqualValues(t, testCase.expectValues, values, testCase.description)
		assert.EqualValues(t, testCase.expectName, actual, testCase.description)
	}
}
