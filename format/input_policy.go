package format

import (
	"fmt"
	"strings"
)

// InputMember describes a parsed object occurrence without losing duplicate order.
// Kind is a JSON kind: object, array, string, number, boolean, or null.
type InputMember struct{ Name, Kind string }
type inputNameRule struct {
	aliases, exclude       []string
	kinds                  map[string]bool
	canonical, alias, fold bool
}

func (t *Tag) HasInputNamePolicy() bool {
	return t != nil && (t.InputNamesPriority != "" || t.InputNamesKinds != "" || t.InputNamesCase != "" || t.InputNamesExclude != "")
}
func parseInputKinds(value string) (map[string]bool, error) {
	r := map[string]bool{}
	for _, v := range strings.Split(value, "|") {
		switch v {
		case "object", "array", "string", "number", "boolean", "null":
		default:
			return nil, fmt.Errorf("unsupported inputNamesKinds %q", v)
		}
		if r[v] {
			return nil, fmt.Errorf("duplicate input kind %q", v)
		}
		r[v] = true
	}
	return r, nil
}
func (m *InputNameMatcher) compilePolicies(fields []InputName, policies []*Tag) error {
	if len(policies) > 0 && len(policies) != len(fields) {
		return fmt.Errorf("input name policy count must match selected fields")
	}
	m.rules = make([]inputNameRule, len(fields))
	for i, field := range fields {
		r := &m.rules[i]
		if field.Aliases != "" {
			r.aliases, _ = parseInputNames(field.Aliases)
		}
		if len(policies) == 0 || policies[i] == nil {
			continue
		}
		p := policies[i]
		if p.HasInputNamePolicy() && field.Aliases == "" {
			return fmt.Errorf("input name policies require aliases for %s", field.Name)
		}
		if p.InputNamesPriority != "" && p.InputNamesPriority != "canonical" && p.InputNamesPriority != "alias" {
			return fmt.Errorf("unsupported input name priority")
		}
		if p.InputNamesCase != "" && p.InputNamesCase != "fold" {
			return fmt.Errorf("unsupported input name case")
		}
		r.canonical = p.InputNamesPriority == "canonical"
		r.alias = p.InputNamesPriority == "alias"
		r.fold = p.InputNamesCase == "fold"
		m.policies = m.policies || p.HasInputNamePolicy()
		if p.InputNamesKinds != "" {
			var err error
			r.kinds, err = parseInputKinds(p.InputNamesKinds)
			if err != nil {
				return err
			}
		}
		if p.InputNamesExclude != "" {
			var err error
			r.exclude, err = parseInputNames(p.InputNamesExclude)
			if err != nil {
				return err
			}
			for _, excluded := range r.exclude {
				matched := false
				for _, alias := range r.aliases {
					matched = matched || alias == excluded || r.fold && strings.EqualFold(alias, excluded)
				}
				if !matched {
					return fmt.Errorf("excluded name %q is not an alias of %s", excluded, field.Name)
				}
				for _, canonical := range m.names {
					if strings.EqualFold(excluded, canonical) {
						return fmt.Errorf("excluded alias %q conflicts with canonical field", excluded)
					}
				}
			}
		}
	}
	for i, left := range m.rules {
		for j := i + 1; j < len(m.rules); j++ {
			right := m.rules[j]
			for _, a := range left.aliases {
				for _, b := range right.aliases {
					if a == b || (left.fold || right.fold) && strings.EqualFold(a, b) {
						return fmt.Errorf("ambiguous input aliases %q and %q", a, b)
					}
				}
			}
		}
	}
	return nil
}

// HasPolicies distinguishes explicit admission policies from legacy exact aliases.
func (m *InputNameMatcher) HasPolicies() bool { return m != nil && m.policies }

// Classify returns the selected field index, whether its spelling is an alias,
// and whether the spelling is explicitly excluded. Unknown names have index -1.
func (m *InputNameMatcher) Classify(name string) (int, bool, bool) {
	// Canonical ownership always wins; aliases cannot hide selected fields.
	for i, canonical := range m.names {
		if name == canonical {
			return i, false, false
		}
	}
	for i, canonical := range m.names {
		if strings.EqualFold(name, canonical) {
			return i, false, false
		}
	}
	for _, r := range m.rules {
		for _, excluded := range r.exclude {
			if name == excluded {
				return -1, true, true
			}
		}
	}
	for i, r := range m.rules {
		for _, alias := range r.aliases {
			if name == alias || r.fold && strings.EqualFold(name, alias) {
				return i, true, false
			}
		}
	}
	return -1, false, false
}

// Select returns admission for each parsed occurrence. Unknown members remain
// admitted for the decoder's own unknown-field policy. Discarded aliases never
// change canonical data or presence. Duplicate occurrences retain their order.
func (m *InputNameMatcher) Select(members []InputMember) []bool {
	result := make([]bool, len(members))
	canonical := make([]bool, len(m.names))
	aliases := make([]bool, len(m.names))
	for _, member := range members {
		index, alias, excluded := m.Classify(member.Name)
		if index >= 0 && !alias && !excluded {
			canonical[index] = true
		}
		if index >= 0 && alias && !excluded {
			r := m.rules[index]
			if len(r.kinds) == 0 || r.kinds[member.Kind] {
				aliases[index] = true
			}
		}
	}
	for i, member := range members {
		index, alias, excluded := m.Classify(member.Name)
		result[i] = !excluded
		if excluded || index < 0 {
			continue
		}
		r := m.rules[index]
		if !alias {
			result[i] = !r.alias || !aliases[index]
			continue
		}
		result[i] = (!r.canonical || !canonical[index]) && (len(r.kinds) == 0 || r.kinds[member.Kind])
	}
	return result
}
