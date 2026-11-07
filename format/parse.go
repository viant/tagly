package format

import (
	"fmt"
	ftime "github.com/viant/tagly/format/time"
	"strings"
	"time"
)

// ParseTime parses time
func (t *Tag) ParseTime(value string) (time.Time, error) {
	if t.TimeInputLayouts != "" {
		for _, layout := range strings.Split(t.TimeInputLayouts, "|") {
			if parsed, err := time.Parse(layout, value); err == nil {
				return *t.adjustTimezone(&parsed), nil
			}
		}
		return time.Time{}, fmt.Errorf("time value %q does not match the declared input layouts", value)
	}
	return ftime.Parse(t.TimeLayout, value)
}
