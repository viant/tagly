package format

import "testing"

func TestInputNameMatcherExactAliasesAndCanonicalFolding(t *testing.T) {
	m, err := NewInputNameMatcher([]InputName{{Name: "metric25", Aliases: "metric_25"}, {Name: "otherValue", Aliases: "other_value"}})
	if err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]int{"metric25": 0, "METRIC25": 0, "metric_25": 0, "METRIC_25": -1, "other_value": 1, "OTHERVALUE": 1, "unknown": -1} {
		if got := m.Match(key); got != want {
			t.Errorf("%q: got %d want %d", key, got, want)
		}
	}
}

func TestInputNameMatcherRejectsAmbiguousAliases(t *testing.T) {
	for _, fields := range [][]InputName{
		{{Name: "one", Aliases: "shared"}, {Name: "two", Aliases: "shared"}},
		{{Name: "one", Aliases: "two"}, {Name: "two"}},
		{{Name: "one", Aliases: "TWO"}, {Name: "two"}},
		{{Name: "one", Aliases: "one|"}},
		{{Name: "one", Aliases: "alias|alias"}},
	} {
		if _, err := NewInputNameMatcher(fields); err == nil {
			t.Errorf("ambiguous or malformed names accepted: %+v", fields)
		}
	}
}

func TestInputNameMatcherPreservesCanonicalDominanceAndOwnsInput(t *testing.T) {
	fields := []InputName{{Name: "Name"}, {Name: "name", Aliases: "snake_name"}}
	m, err := NewInputNameMatcher(fields)
	if err != nil {
		t.Fatal(err)
	}
	fields[0].Name = "mutated"
	for key, want := range map[string]int{"Name": 0, "name": 1, "NAME": 0, "snake_name": 1, "SNAKE_NAME": -1, "mutated": -1} {
		if got := m.Match(key); got != want {
			t.Errorf("%q: got %d want %d", key, got, want)
		}
	}
}
