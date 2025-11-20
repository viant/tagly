package format

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse_DataDriven(t *testing.T) {
	var testCases = []struct {
		description string
		tag         reflect.StructTag
		names       []string
		expect      Tag
		wantErr     bool
	}{
		{
			description: "name only",
			tag:         reflect.StructTag(`format:"Id"`),
			expect:      Tag{Name: "Id"},
		},
		{
			description: "unknown key in strict mode fails",
			tag:         reflect.StructTag(`format:"unknown=abc"`),
			wantErr:     true,
		},
		{
			description: "fallback json simple name with omitempty influences Omitempty",
			tag:         reflect.StructTag(`format:"dateFormat=YYYY-MM-DD,name=startDate" json:"Id,omitempty"`),
			names:       []string{"json"},
			// Implementation prefers fallback simple name if provided; we assert Omitempty and layout only.
			expect: Tag{Omitempty: true, TimeLayout: "2006-01-02"},
		},
		{
			description: "ignore case formatter '-'",
			tag:         reflect.StructTag(`format:"-"`),
			expect:      Tag{CaseFormat: "-"},
		},
		{
			description: "timezone set to UTC",
			tag:         reflect.StructTag(`format:"timezone=UTC"`),
			expect:      Tag{Timezone: "UTC"},
		},
		{
			description: "nullable true",
			tag:         reflect.StructTag(`format:"nullable=true"`),
			expect:      func() Tag { b := true; return Tag{Nullable: &b} }(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			got, err := Parse(tc.tag, tc.names...)
			if tc.wantErr {
				assert.NotNil(t, err, tc.description)
				return
			}
			assert.Nil(t, err, tc.description)
			if tc.expect.Nullable != nil && got.Nullable != nil {
				assert.Equal(t, *tc.expect.Nullable, *got.Nullable)
				// align to compare remaining fields
				b := *tc.expect.Nullable
				tc.expect.Nullable = &b
			}
			// compare selected fields
			if tc.expect.Name != "" {
				assert.Equal(t, tc.expect.Name, got.Name)
			}
			assert.Equal(t, tc.expect.CaseFormat, got.CaseFormat)
			assert.Equal(t, tc.expect.Omitempty, got.Omitempty)
			assert.Equal(t, tc.expect.Timezone, got.Timezone)
			assert.Equal(t, tc.expect.TimeLayout, got.TimeLayout)
			if tc.expect.Nullable != nil || got.Nullable != nil {
				assert.Equal(t, tc.expect.Nullable != nil, got.Nullable != nil)
				if tc.expect.Nullable != nil && got.Nullable != nil {
					assert.Equal(t, *tc.expect.Nullable, *got.Nullable)
				}
			}
		})
	}
}

func TestIsValidTagKey(t *testing.T) {
	assert.True(t, IsValidTagKey("name"))
	assert.True(t, IsValidTagKey("inline"))
	assert.False(t, IsValidTagKey("nonexistent"))
}
