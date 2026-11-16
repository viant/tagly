package choice

// IsEnumTag reports whether a struct tag key configures the enum codec.
// Transport markers, such as mcpEnum, are owned by their respective callers.
func IsEnumTag(name string) bool {
	switch name {
	case "choice", "enumAccept", "enumStyle", "enumCase", "enumTrim",
		"enumEmpty", "enumZero", "enumNull", "enumInteger", "enumUnknownInteger",
		"enumUnknownString", "enumAlias", "enumOutputAlias", "enumOutputUnknown":
		return true
	}
	return false
}
