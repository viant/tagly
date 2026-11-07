package format

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// TimeAction tells a native decoder how to apply a declared time value.
// Preserve retains an existing value; a fresh pointer is allocated as Go zero.
type TimeAction uint8

const (
	TimeAssign TimeAction = iota
	TimePreserve
	TimeNull
)

// HasTimeInputPolicy reports an explicit input policy, leaving untagged time
// conversion under its existing owner's compatibility rules.
func (t *Tag) HasTimeInputPolicy() bool {
	return t != nil && (t.TimeInputLayouts != "" || t.TimeEpoch != "" || t.TimeEmpty != "")
}

// ParseTimeJSON parses one original JSON scalar without losing integer spelling
// or duplicate-member history. The caller owns presence and pointer assignment.
func (t *Tag) ParseTimeJSON(raw []byte) (time.Time, TimeAction, error) {
	raw = bytes.TrimSpace(raw)
	if bytes.Equal(raw, []byte("null")) {
		return time.Time{}, TimeNull, nil
	}
	if len(raw) > 0 && raw[0] == '"' {
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return time.Time{}, TimeAssign, err
		}
		if value == "" && t.TimeEmpty == "preserve" {
			return time.Time{}, TimePreserve, nil
		}
		stamp, err := t.ParseTime(value)
		return stamp, TimeAssign, err
	}
	if t.TimeEpoch == "milliseconds" && json.Valid(raw) {
		millis, err := strconv.ParseInt(string(raw), 10, 64)
		if err == nil {
			return time.UnixMilli(millis).UTC(), TimeAssign, nil
		}
	}
	return time.Time{}, TimeAssign, fmt.Errorf("time value must match the declared string or integer epoch policy")
}
