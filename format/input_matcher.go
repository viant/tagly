package format

import (
	"fmt"
	"strings"
)

// InputName describes a field already selected by the consumer's canonical
// visibility and dominance rules. Aliases is Tag.InputNames; it is decode-only.
type InputName struct {
	Name    string
	Aliases string
}

// InputNameMatcher resolves admitted object members to canonical field indexes.
// It owns its metadata and is safe for concurrent use after construction.
type InputNameMatcher struct {
	names    []string
	exact    map[string]int
	rules    []inputNameRule
	policies bool
}

// NewInputNameMatcher retains canonical exact/folded lookup, adding only exact
// aliases. The caller must supply the selected fields in canonical lookup order;
// this matcher does not select fields or grant visibility to omitted fields.
func NewInputNameMatcher(fields []InputName, policies ...*Tag) (*InputNameMatcher, error) {
	m := &InputNameMatcher{names: make([]string, len(fields)), exact: make(map[string]int)}
	for i, field := range fields {
		m.names[i] = field.Name
		if _, exists := m.exact[field.Name]; !exists {
			m.exact[field.Name] = i
		}
	}
	for i, field := range fields {
		if field.Aliases == "" {
			continue
		}
		aliases, err := parseInputNames(field.Aliases)
		if err != nil {
			return nil, err
		}
		for _, alias := range aliases {
			for j, name := range m.names {
				if i != j && strings.EqualFold(alias, name) {
					return nil, fmt.Errorf("input alias %q for %q conflicts with canonical field %q", alias, field.Name, name)
				}
			}
			if owner, exists := m.exact[alias]; exists && owner != i {
				return nil, fmt.Errorf("input alias %q ambiguously addresses %q and %q", alias, m.names[owner], field.Name)
			}
			m.exact[alias] = i
		}
	}
	if err := m.compilePolicies(fields, policies); err != nil {
		return nil, err
	}
	return m, nil
}

// Match returns the canonical field index, or -1 for an unknown member. Alias
// spelling is exact; only canonical names participate in case-folded matching.
func (m *InputNameMatcher) Match(name string) int {
	if m == nil {
		return -1
	}
	if m.policies {
		index, _, excluded := m.Classify(name)
		if excluded {
			return -1
		}
		return index
	}
	if index, ok := m.exact[name]; ok {
		return index
	}
	for i, canonical := range m.names {
		if strings.EqualFold(name, canonical) {
			return i
		}
	}
	return -1
}

func parseInputNames(value string) ([]string, error) {
	names := strings.Split(value, "|")
	seen := make(map[string]bool, len(names))
	for _, name := range names {
		if name == "" {
			return nil, fmt.Errorf("inputNames contains an empty alias")
		}
		if seen[name] {
			return nil, fmt.Errorf("inputNames contains duplicate alias %q", name)
		}
		seen[name] = true
	}
	return names, nil
}
