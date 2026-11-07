package format

import (
	"fmt"
	"go/token"
	"strings"
	"unicode"
	"unicode/utf8"
)

// InputGuard suppresses a field when an admitted ancestor Go field is present.
// Depth counts enclosing struct objects, not pointers or collection containers.
type InputGuard struct {
	Depth int
	Field string
}

// InputScope declares visible Go fields and their admitted presence. Null is present.
type InputScope map[string]bool

func parseInputGuard(value string) (InputGuard, error) {
	result := InputGuard{}
	for strings.HasPrefix(value, "../") {
		result.Depth++
		value = strings.TrimPrefix(value, "../")
	}
	first, _ := utf8.DecodeRuneInString(value)
	if result.Depth == 0 || !token.IsIdentifier(value) || !unicode.IsUpper(first) {
		return result, fmt.Errorf("inputUnless requires ../ followed by an exported Go field name")
	}
	result.Field = value
	return result, nil
}
func (t *Tag) InputGuard() (InputGuard, error) {
	if t == nil || t.InputUnless == "" {
		return InputGuard{}, nil
	}
	return parseInputGuard(t.InputUnless)
}
func (m *InputNameMatcher) Guards() []InputGuard {
	if m == nil {
		return nil
	}
	result := make([]InputGuard, len(m.rules))
	for i, r := range m.rules {
		result[i] = r.guard
	}
	return result
}
func ValidateInputGuardContext(guards []InputGuard, ancestors []InputScope) error {
	for _, guard := range guards {
		if guard.Depth < 0 || (guard.Depth == 0 && guard.Field != "") {
			return fmt.Errorf("invalid inputUnless compiled guard")
		}
		if guard.Depth == 0 {
			continue
		}
		if guard.Depth > len(ancestors) {
			return fmt.Errorf("inputUnless ancestor depth %d unavailable for %s", guard.Depth, guard.Field)
		}
		if _, ok := ancestors[guard.Depth-1][guard.Field]; !ok {
			return fmt.Errorf("inputUnless target %s is not a visible ancestor field", guard.Field)
		}
	}
	return nil
}

// SelectScoped excludes ancestor-guarded candidates before ordinary alias/group selection.
// Ancestors are nearest-first. Caller plans validate contexts before decoding.
func (m *InputNameMatcher) SelectScoped(members []InputMember, ancestors []InputScope) ([]bool, error) {
	if err := ValidateInputGuardContext(m.Guards(), ancestors); err != nil {
		return nil, err
	}
	result := make([]bool, len(members))
	admitted := make([]InputMember, 0, len(members))
	positions := make([]int, 0, len(members))
	for i, member := range members {
		index, _, _ := m.Classify(member.Name)
		if index >= 0 {
			guard := m.rules[index].guard
			if guard.Depth > 0 && ancestors[guard.Depth-1][guard.Field] {
				continue
			}
		}
		admitted = append(admitted, member)
		positions = append(positions, i)
	}
	selected := m.Select(admitted)
	for i, keep := range selected {
		result[positions[i]] = keep
	}
	return result, nil
}

// InputScopePlan is an immutable compiled shape graph. A nil Scope denotes a
// transparent pointer/collection container; object scopes contain visible fields.
// Owners construct it while compiling their existing field plans.
type InputScopePlan struct {
	Scope    InputScope
	Guards   []InputGuard
	Children []*InputScopePlan
	Element  *InputScopePlan
}

// ValidateInputScopePlan checks every possible type context, including recursive
// shapes. Only the bounded ancestor suffix referenced by guards affects identity.
func ValidateInputScopePlan(root *InputScopePlan) error {
	maxDepth := 0
	seen := map[*InputScopePlan]bool{}
	var scan func(*InputScopePlan)
	scan = func(p *InputScopePlan) {
		if p == nil || seen[p] {
			return
		}
		seen[p] = true
		for _, g := range p.Guards {
			if g.Depth > maxDepth {
				maxDepth = g.Depth
			}
		}
		scan(p.Element)
		for _, c := range p.Children {
			scan(c)
		}
	}
	scan(root)
	if maxDepth == 0 {
		return nil
	}
	visited := map[string]bool{}
	var visit func(*InputScopePlan, []*InputScopePlan) error
	visit = func(p *InputScopePlan, ancestors []*InputScopePlan) error {
		if p == nil {
			return nil
		}
		signature := fmt.Sprintf("%p", p)
		for i, a := range ancestors {
			if i >= maxDepth {
				break
			}
			signature += fmt.Sprintf("/%p", a)
		}
		if visited[signature] {
			return nil
		}
		visited[signature] = true
		if p.Element != nil {
			return visit(p.Element, ancestors)
		}
		next := ancestors
		if p.Scope != nil {
			scopes := make([]InputScope, len(ancestors))
			for i, a := range ancestors {
				scopes[i] = a.Scope
			}
			if err := ValidateInputGuardContext(p.Guards, scopes); err != nil {
				return err
			}
			next = append([]*InputScopePlan{p}, ancestors...)
			if len(next) > maxDepth {
				next = next[:maxDepth]
			}
		}
		for _, c := range p.Children {
			if err := visit(c, next); err != nil {
				return err
			}
		}
		return nil
	}
	return visit(root, nil)
}
