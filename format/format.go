package format

import (
	"github.com/viant/tagly/format/text"
	"time"
)

// FormatTime formats time
func (t *Tag) FormatTime(ts *time.Time) string {

	ts = t.adjustTimezone(ts)

	if t.TimeLayout == "" {
		return ts.Format(time.RFC3339)
	}
	return ts.Format(t.TimeLayout)
}

func (t *Tag) adjustTimezone(ts *time.Time) *time.Time {
	if t.Timezone == "" {
		return ts
	}
	switch t.Timezone {
	case "utc", "UTC":
		inZone := ts.In(time.UTC)
		ts = &inZone
	default:
		if t.tz == nil {
			t.tz, _ = time.LoadLocation(t.Timezone)
		}
		if tz := t.tz; tz != nil {
			inZone := ts.In(tz)
			ts = &inZone
		}
	}
	return ts
}

// FormatName formata name
func (t *Tag) FormatName() string {
	return t.CaseFormatName("")
}

// CaseFormatName returns case formatted name
func (t *Tag) CaseFormatName(defaultCaseFormat text.CaseFormat) string {
	if t.CaseFormat == "-" {
		return t.Name
	}
	caseFormat := t.CaseFormat
	if caseFormat == "" {
		caseFormat = string(defaultCaseFormat)
	}
	if caseFormat == "" {
		return t.Name
	}

	if t.formatter != nil {
		if string(t.formatter.To()) != caseFormat {
			t.formatter = nil
		}
	}
	if t.formatter == nil {
		to := text.NewCaseFormat(caseFormat)
		t.formatter = text.CaseFormatUpperCamel.To(to)
		t.CaseFormat = string(to)
	}
	return t.formatter.Format(t.Name)
}
