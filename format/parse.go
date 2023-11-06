package format

import (
	ftime "github.com/viant/tagly/format/time"
	"time"
)

func (t *Tag) ParseTime(value string) (time.Time, error) {
	return ftime.Parse(t.TimeLayout, value)
}
