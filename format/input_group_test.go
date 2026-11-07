package format

import (
	"reflect"
	"testing"
)

func groupMatcher(t *testing.T) *InputNameMatcher {
	t.Helper()
	fields := []InputName{{Name: "day_of_week", Aliases: "dayOfWeek"}, {Name: "Weekday"}, {Name: "weekday"}}
	var policies []*Tag
	for _, tag := range []reflect.StructTag{`format:"inputGroup=weekday,inputPriority=0,inputNames=dayOfWeek,inputNamesPriority=alias"`, `format:"inputGroup=weekday,inputPriority=1,inputCase=exact"`, `format:"inputGroup=weekday,inputPriority=2,inputCase=exact"`} {
		p, e := Parse(tag)
		if e != nil {
			t.Fatal(e)
		}
		policies = append(policies, p)
	}
	m, e := NewInputNameMatcher(fields, policies...)
	if e != nil {
		t.Fatal(e)
	}
	return m
}
func TestInputGroupSelection(t *testing.T) {
	m := groupMatcher(t)
	for _, tc := range []struct {
		names []string
		want  []bool
	}{
		{[]string{"Weekday", "weekday"}, []bool{false, true}},
		{[]string{"weekday", "Weekday"}, []bool{true, false}},
		{[]string{"day_of_week", "Weekday", "weekday"}, []bool{false, false, true}},
		{[]string{"Weekday", "Weekday"}, []bool{true, true}},
		{[]string{"day_of_week", "dayOfWeek"}, []bool{false, true}},
	} {
		members := make([]InputMember, len(tc.names))
		for i, n := range tc.names {
			members[i] = InputMember{n, "null"}
		}
		if got := m.Select(members); !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("%v: %v want%v", tc.names, got, tc.want)
		}
	}
	for _, name := range []string{"WEEKDAY", "weekDay"} {
		if got := m.Match(name); got != -1 {
			t.Fatalf("forbidden case %s matched%d", name, got)
		}
	}
	if m.Match("DAY_OF_WEEK") != 0 {
		t.Fatal("ordinary canonical folding changed")
	}
}
func TestInputGroupInvalidConfiguration(t *testing.T) {
	for _, tag := range []reflect.StructTag{`format:"inputGroup=x"`, `format:"inputPriority=0"`, `format:"inputGroup=x,inputPriority=-1"`, `format:"inputGroup=x,inputPriority=4294967296"`, `format:"inputCase=fold"`} {
		p, err := Parse(tag)
		if err == nil {
			_, err = NewInputNameMatcher([]InputName{{Name: "a"}}, p)
		}
		if err == nil {
			t.Fatalf("accepted %s", tag)
		}
	}
	p, err := Parse(`format:"inputGroup=x,inputPriority=0"`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = NewInputNameMatcher([]InputName{{Name: "a"}, {Name: "b"}}, p, p); err == nil {
		t.Fatal("priority tie accepted")
	}
	if _, err = NewInputNameMatcher([]InputName{{Name: "a"}}, p); err != nil {
		t.Fatal("singleton group must survive visibility", err)
	}
}

func TestInputGroupsComposeWithAliasAdmission(t *testing.T) {
	fields := []InputName{{Name: "base"}, {Name: "preferred", Aliases: "old"}, {Name: "other"}}
	tags := []reflect.StructTag{`format:"inputGroup=a,inputPriority=0"`, `format:"inputGroup=a,inputPriority=1,inputNames=old,inputNamesKinds=object|null"`, `format:"inputGroup=b,inputPriority=0"`}
	policies := make([]*Tag, len(tags))
	for i, tag := range tags {
		var err error
		policies[i], err = Parse(tag)
		if err != nil {
			t.Fatal(err)
		}
	}
	m, err := NewInputNameMatcher(fields, policies...)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		kind string
		want []bool
	}{{"number", []bool{true, false, true}}, {"null", []bool{false, true, true}}} {
		got := m.Select([]InputMember{{"base", "number"}, {"old", tc.kind}, {"other", "string"}})
		if !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("kind%s got%v", tc.kind, got)
		}
	}
}

func TestInputExactCanonicalVariantExclusion(t *testing.T) {
	p, err := Parse(`format:"inputNames=status,inputNamesExclude=status"`)
	if err != nil {
		t.Fatal(err)
	}
	m, err := NewInputNameMatcher([]InputName{{Name: "Status", Aliases: p.InputNames}}, p)
	if err != nil {
		t.Fatal(err)
	}
	if m.Match("status") != -1 || m.Select([]InputMember{{"status", "object"}})[0] {
		t.Fatal("excluded variant admitted")
	}
	for _, name := range []string{"Status", "STATUS", "sTaTuS"} {
		if m.Match(name) != 0 {
			t.Fatalf("accepted spelling lost %s", name)
		}
	}
	p.InputNamesExclude = "Status"
	p.InputNames = "Status"
	if _, err = NewInputNameMatcher([]InputName{{Name: "Status", Aliases: "Status"}}, p); err == nil {
		t.Fatal("exact canonical exclusion accepted")
	}
}
