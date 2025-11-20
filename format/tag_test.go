package format

import (
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {

	var testCases = []struct {
		description string
		tag         reflect.StructTag
		tagName     string
	}{

		{
			description: "fallback",
			tagName:     "xjson",
			tag:         reflect.StructTag(`format:"dateFormat=YYYY-MM-DD,name=startDate" xjson:"dateFormat=YYYY-MM-DD-hh:mm" `),
		},

		{
			description: "fallback simple name ",
			tagName:     "json",
			tag:         reflect.StructTag(`format:"dateFormat=YYYY-MM-DD,name=startDate" json:"Id,omitempty" `),
		},
	}

	for _, testCase := range testCases {
		_, err := Parse(testCase.tag, testCase.tagName)
		assert.Nil(t, err)
	}
}
