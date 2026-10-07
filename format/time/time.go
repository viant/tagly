package time

import (
	"strings"
	"time"
)

var iso20220715DateFormatToRfc3339TimeLayoutReplacer = strings.NewReplacer(
	"YYYY", "2006",
	"yyyy", "2006",
	"MM", "01",
	"M", "1",
	"DD", "02",
	"dd", "02",
	"D", "2",
	"+hh:mm", "Z07:00",
	"+hhmm", "Z0700",
	"+hh", "Z07",
	"-hh:mm", "Z07:00",
	"-hhmm", "Z0700",
	"hh", "15",
	"mm", "04",
	"m", "4",
	"ss", "05",
	".SSS", ".999",
	".SS", ".99",
	".S", ".9",
	"-hh", "Z07",
	"Z", "Z07:00",
)

var rfc3339TimeLayoutToIso20220715DateFormatReplacer = strings.NewReplacer(
	// Year
	"2006", "YYYY",

	// Time components
	"15", "hh",
	"04", "mm",
	"05", "ss",

	// Fractional seconds
	".999", ".SSS",
	".99", ".SS",
	".9", ".S",

	// Month/Day (order matters: longer tokens first)
	"01", "MM",
	"02", "DD",
	"1", "M",
	"2", "D",
	"4", "m",

	// Time zone: map RFC3339 layout to generic 'Z' token
	// Use longest first to avoid partial matches
	"Z07:00", "Z",
	"Z0700", "Z",
	"Z07", "Z",
)

// DateFormatToTimeLayout converts ISO 2022-07-15 date format to RFC3339 time layout
func DateFormatToTimeLayout(dateFormat string) string {
	return iso20220715DateFormatToRfc3339TimeLayoutReplacer.Replace(dateFormat)
}

// TimeLayoutToDateFormat converts RFC3339 time layout to ISO 2022-07-15 date format
func TimeLayoutToDateFormat(dateFormat string) string {
	return rfc3339TimeLayoutToIso20220715DateFormatReplacer.Replace(dateFormat)
}

func Parse(layout, value string) (time.Time, error) {
	if layout == "" {
		layout = time.RFC3339 //TODO add layout autodetection for this case
	}

	if strings.Contains(value, ".") {
		if layout == time.RFC3339 {
			layout = time.RFC3339Nano
		}
	}

	//adjust T fragment
	if strings.Contains(value, "T") != strings.Contains(layout, "T") {
		layout = strings.Replace(layout, "T", " ", 1)
		value = strings.Replace(value, "T", " ", 1)
	}
	// If layout expects timezone but value does not provide one, try without timezone
	if strings.Contains(layout, "Z07") && !containsTZ(value) {
		layout = strings.Replace(layout, "Z07:00", "", 1)
		layout = strings.Replace(layout, "Z0700", "", 1)
		layout = strings.Replace(layout, "Z07", "", 1)
		layout = strings.TrimSpace(layout)
	}
	t, err := time.ParseInLocation(layout, value, time.UTC)

	originalLayout := layout

	if err != nil {

		if len(value) > len(layout) {
			value = value[:len(layout)]
			t, err = time.Parse(layout, value)
		} else {
			layout = layout[:len(value)]
			t, err = time.Parse(layout, value)
		}

		parseErr, ok := err.(*time.ParseError)
		if ok && err != nil {
			if parseErr.LayoutElem != "" {
				if index := strings.LastIndex(originalLayout, parseErr.LayoutElem); index != -1 {
					value = value[:index]
					layout = originalLayout[:index]
					t, err = time.Parse(layout, value)
				}
			}
		}

		if err != nil {
			layout = "2006-01-02-07:00"
			t, err = time.Parse(layout, value)
		}

		// Final pragmatic fallback: try basic human layout without TZ if still failing
		if err != nil {
			if t2, err2 := time.ParseInLocation("2006-01-02 15:04:05", value, time.UTC); err2 == nil {
				return t2, nil
			}
		}
	}
	return t, err
}

// containsTZ reports whether value contains an explicit timezone suffix (Z or ±hh[:]mm)
func containsTZ(value string) bool {
	if strings.Contains(value, "Z") {
		return true
	}
	// check last ~6 chars for + or - (to avoid date hyphens)
	n := len(value)
	start := n - 6
	if start < 0 {
		start = 0
	}
	tail := value[start:]
	return strings.Contains(tail, "+") || strings.Contains(tail, "-")
}
