package text

import (
	"strings"
	"sync"
)

func FormatName(name, to string) string {
	to = normalizeCase(to)
	cf := NewCaseFormat(strings.ToLower(to))
	if cf == CaseFormatUndefined {
		return name
	}
	return DetectCaseFormat(name).Format(name, cf)
}

func normalizeCase(to string) string {
	switch strings.ToLower(to) {
	case "snake":
		return "lowerunderscore"
	case "kebab":
		return "lowerdash"
	case "camel":
		return "lowercamel"
	case "pascal":
		return "uppercamel"
	default:
		return to
	}
}

func MakeNameFormatter(to string) func(string) string {
	to = normalizeCase(to)
	cf := NewCaseFormat(strings.ToLower(to))
	if cf == CaseFormatUndefined {
		return func(s string) string { return s }
	}
	var cache sync.Map
	return func(s string) string {
		if v, ok := cache.Load(s); ok {
			return v.(string)
		}
		out := DetectCaseFormat(s).Format(s, cf)
		cache.Store(s, out)
		return out
	}
}
