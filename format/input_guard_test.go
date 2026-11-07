package format

import (
	"reflect"
	"testing"
)

func TestInputGuardScopedSelection(t *testing.T) {
	p, err := Parse(`format:"inputUnless=../Canonical"`)
	if err != nil {
		t.Fatal(err)
	}
	m, err := NewInputNameMatcher([]InputName{{Name: "value"}}, p)
	if err != nil {
		t.Fatal(err)
	}
	members := []InputMember{{Name: "value", Kind: "string"}}
	for _, present := range []bool{false, true} {
		got, err := m.SelectScoped(members, []InputScope{{"Canonical": present}})
		if err != nil {
			t.Fatal(err)
		}
		if got[0] == present {
			t.Fatalf("presence %v: %v", present, got)
		}
	}
	for _, scopes := range [][]InputScope{nil, {{"Other": true}}} {
		if _, err := m.SelectScoped(members, scopes); err == nil {
			t.Fatal("invalid context accepted")
		}
	}
	for _, value := range []string{"Canonical", "../", "../hidden.field", "../../X/Y", "../_", "../lower"} {
		if _, err := Parse(reflect.StructTag(`format:"inputUnless=` + value + `"`)); err == nil {
			t.Fatalf("invalid guard %q", value)
		}
	}
}

func TestGuardPrecedesGroupSelection(t *testing.T) {
	low, _ := Parse(`format:"inputGroup=x,inputPriority=0"`)
	high, _ := Parse(`format:"inputGroup=x,inputPriority=1,inputUnless=../Canonical"`)
	m, err := NewInputNameMatcher([]InputName{{Name: "low"}, {Name: "high"}}, low, high)
	if err != nil {
		t.Fatal(err)
	}
	got, err := m.SelectScoped([]InputMember{{Name: "low", Kind: "number"}, {Name: "high", Kind: "object"}}, []InputScope{{"Canonical": true}})
	if err != nil || !got[0] || got[1] {
		t.Fatalf("guard suppressed active group fallback: %v %v", got, err)
	}
	root := &InputScopePlan{Scope: InputScope{"Canonical": false}}
	child := &InputScopePlan{Scope: InputScope{"Value": false}, Guards: []InputGuard{{Depth: 1, Field: "Canonical"}}}
	root.Children = []*InputScopePlan{child}
	child.Children = []*InputScopePlan{child}
	if err := ValidateInputScopePlan(root); err == nil {
		t.Fatal("recursive invalid context accepted")
	}
	child.Scope["Canonical"] = false
	if err := ValidateInputScopePlan(root); err != nil {
		t.Fatal(err)
	}
}

func TestInvalidCompiledGuardContext(t *testing.T) {
	for _, guard := range []InputGuard{{Depth: -1, Field: "Canonical"}, {Field: "Canonical"}} {
		if err := ValidateInputGuardContext([]InputGuard{guard}, []InputScope{{"Canonical": true}}); err == nil {
			t.Fatalf("invalid compiled guard accepted: %#v", guard)
		}
	}
}
