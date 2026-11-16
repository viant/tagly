package choice

import "testing"

func TestIsEnumTag(t *testing.T) {
	for _, key := range []string{"choice", "enumAccept", "enumStyle", "enumCase", "enumTrim", "enumEmpty",
		"enumZero", "enumNull", "enumInteger", "enumUnknownInteger", "enumUnknownString", "enumAlias",
		"enumOutputAlias", "enumOutputUnknown"} {
		if !IsEnumTag(key) {
			t.Errorf("enum key %q was not recognized", key)
		}
	}
	for _, key := range []string{"", "enum", "enumFuture", "EnumAccept", "choiceGroup", "mcpEnum", "internal", "json", "mcp"} {
		if IsEnumTag(key) {
			t.Errorf("unrelated key %q was classified as an enum", key)
		}
	}
}
